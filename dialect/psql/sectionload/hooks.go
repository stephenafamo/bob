package sectionload

import (
	"context"
	"fmt"
	"sort"

	"github.com/stephenafamo/bob"
)

// runThen loads one Then(...) boundary: the edge's child query becomes
// the parent (src) of a new island, and the subtree below it becomes
// that island's sections — the recursive application of the same
// mechanism.
//
// Hook semantics: the child query is composed by us (not via the models
// view), so no child query hooks and no child after-select hooks run.
//
// parentKeyVals is tp.edge's pre-extracted parent-key value per entry of
// parents, aligned positionally with parents. It is nil for the outermost
// call from run.go, which has no scan buffer of its own to have
// precomputed one from (the top-level parent query is the caller's opaque
// bob.Query, not a section execIsland scanned).
func runThen(ctx context.Context, exec bob.Executor, tp thenPlan, parents, parentKeyVals []any) error {
	e := tp.edge
	if parentKeyVals != nil && len(parentKeyVals) != len(parents) {
		return fmt.Errorf("sectionload: %s: %d then parent keys for %d parents", e.relName(), len(parentKeyVals), len(parents))
	}

	// Collect distinct key tuples from the parents (declaration order).
	seen := make(map[any]struct{})
	var keys [][]any
	for i, p := range parents {
		var kv []any
		if parentKeyVals != nil {
			kv, _ = parentKeyVals[i].([]any)
		} else {
			kv = e.parentKeyAny(p)
		}
		if kv == nil {
			continue
		}
		k, err := mapKey(kv)
		if err != nil {
			return fmt.Errorf("sectionload: %s then key: %w", e.relName(), err)
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, kv)
	}

	var children, childKeys []any
	if len(keys) > 0 {
		var err error
		children, childKeys, err = loadThenChildren(ctx, exec, tp, keys)
		if err != nil {
			return err
		}
	}

	// Stitch the child island's roots into this island's parents. children
	// is already in its final order (sorted above when requested), so no
	// ordinal is passed here; the returned (unchanged, since ords is nil)
	// slice is discarded — this call's caller does not read items[e.set]
	// afterward the way stitchIsland's does.
	_, err := stitchInto(e, parents, children, parentKeyVals, childKeys, nil)
	return err
}

// loadThenChildren runs the island opened by one Then(...) boundary for the
// distinct parent key tuples in keys: it builds and executes the child
// query, stitches the island, restores the edge's requested order, runs
// any nested Then boundaries, and returns the island's root rows with the
// key each one exposes as a child of tp.edge.
func loadThenChildren(ctx context.Context, exec bob.Executor, tp thenPlan, keys [][]any) ([]any, []any, error) {
	e := tp.edge
	// edgeRootParts, not e.child() directly: e's ColumnPlan restricts
	// which columns this island's root query selects, so this
	// island's root must report the same restricted column list, not
	// the child table's unrestricted TableParts.Columns (see
	// edgeRootParts's doc; its cols() already returns
	// e.childCols()).
	childRoot := edgeRootParts{e.child(), e}
	// Route 4 of buildEdgeSelect: the Then boundary's parent query, matched
	// against the parent keys collected above. The edge's own options apply
	// here too, same as any other route. collapseOrderBy is true because this
	// query becomes the new island's own root, and composeIsland's _sl_src
	// wrapping needs a real ordinal column to ORDER BY explicitly rather than
	// a bare literal ORDER BY it can only hope the planner preserves through
	// an extra row_number() OVER () with no ORDER BY of its own. When the edge
	// requested no ORDER BY, collapseOrderBy is a no-op here exactly as at
	// routes 1/2.
	// edgeJoinChain(e), not childRoot.cols(): for a multi-hop e, the
	// physical SELECT below reads the final table's own columns
	// (childRoot.scanCols(), delegating to e.childScanCols()) plus the
	// carried key(s) via a JOIN through e's intermediate table(s) — the same
	// route-1/2 treatment, since this IS a physical SELECT.
	// childRoot.cols() (== e.childCols(), scan+carried) stays the
	// layout-level column list composeIsland below uses for the new island's
	// own root.
	joins, carried, matchAlias := edgeJoinChain(e)

	// opts/fragment/ordered are extracted exactly once here, before
	// parentQuery is built, and shared by buildEdgeSelect (via
	// edgeSelectSpec.fragment, below), by parentOrdCol's decision, and by the
	// post-sort check further down this function. Extracting them
	// independently at each of those three sites (hasOrderBy(e.options()) or,
	// for buildEdgeSelect, its own internal extraction) could make them
	// disagree for a stateful bob.ModFunc (e.g. a mod that only adds ORDER BY
	// on every other call): parentQuery might carry a real _sl_row_ord
	// ordinal while parentOrdCol stays "" (so composeIsland's _sl_src never
	// reads it back), or the post-sort step might run (or skip) based on a
	// different evaluation than the one that actually decided whether
	// parentQuery has an ORDER BY at all.
	opts := e.options()
	var fragment *permittedFragment
	var ordered bool
	if len(opts.mods) > 0 {
		frag, err := extractOptionsFragment(opts, e.relName())
		if err != nil {
			return nil, nil, err
		}
		fragment = &frag
		ordered = len(frag.orderBy.Expressions) > 0
	}

	parentQuery, err := buildEdgeSelect(edgeSelectSpec{
		edge:            e,
		cols:            childRoot.scanCols(),
		from:            tableExpr(childRoot.schemaName(), childRoot.tableName()),
		alias:           srcAlias,
		joins:           joins,
		carried:         carried,
		matchAlias:      matchAlias,
		keys:            keys,
		opts:            opts,
		collapseOrderBy: true,
		fragment:        fragment,
	})
	if err != nil {
		return nil, nil, err
	}
	childPlan, err := planIsland(tp.children)
	if err != nil {
		return nil, nil, err
	}
	parentOrdCol := ""
	if ordered {
		parentOrdCol = ordCol
	}
	// composeIsland wraps its first argument in parentSource, which checks
	// the RENDERED SQL for sectionload's reserved identifiers — a check that
	// exists to catch a CALLER's parent query colliding with sectionload's
	// own namespace. parentQuery here is never that: it is sectionload's OWN
	// buildEdgeSelect output, and for a multi-hop e it legitimately SELECTs
	// its own carried-key column(s) under exactly the "_sl_k<i>" names that
	// check protects (a multi-hop edge placed behind a real Then(...)
	// boundary renders a route-4 query the check itself would otherwise
	// reject). trustedParentQuery (parentexpr.go) exempts it from that one
	// check while keeping parentSource's single-render guarantee — see its
	// own doc for why exempting the WHOLE query is safe here. Scoped to
	// len(e.sides())>1 only: a single-hop edge's route-4 query never contains
	// a carried column and so never needs the exemption, keeping the check's
	// original protection (e.g. against a caller-supplied EdgeWhere.SQLValue
	// raw fragment that happened to contain a reserved-looking substring)
	// exactly as before for every other edge.
	var composedParent bob.Query = parentQuery
	if len(e.sides()) > 1 {
		composedParent = trustedParentQuery{parentQuery}
	}
	q, err := composeIsland(composedParent, childRoot, childPlan, parentOrdCol)
	if err != nil {
		return nil, nil, err
	}
	// e.childKeyFrom, not e.childKeyFromBufAny: identical for a single-hop e
	// (childKeyFrom just calls childKeyFromBufAny), but for a multi-hop e the
	// root row's carried value (scanned by execIsland's own rootCarriedBuf,
	// driven by childRoot's carriedCols()/newCarriedBufAny() delegating to e
	// — see edgeRootParts in node.go) is what must be read instead.
	res, err := execIsland(ctx, exec, q, childRoot, childPlan, e.childKeyFrom)
	if err != nil {
		return nil, nil, err
	}
	if err := stitchIsland(childPlan, res); err != nil {
		return nil, nil, err
	}
	children := res.items[0]
	// rootChildKeys, not parentKeys[0]: 0 is never a section's own set
	// number, so parentKeys (keyed by section set) never has an entry for it
	// — execIsland's rootChildKeyFn parameter exists to compute exactly this
	// value instead (reading parentKeys[0] would silently get nil and fall
	// back to a model-based ChildKey lookup, which breaks once the key column
	// is Internal-only).
	childKeys := res.rootChildKeys

	// Restore the edge's requested order: composeIsland's own _sl_src
	// row_number() OVER () wrapping already captured it in res.ords (the same
	// mechanism every plain root query relies on), but unlike run.go's
	// top-level All(), nothing has applied it to children yet. Skipped
	// entirely when the edge asked for no ORDER BY.
	//
	// items[0], childKeys, and res.thenKeys (for any Then(...) nested
	// directly under this island's root) are all reassigned together,
	// mirroring run.go's own "Keep items[0] in the restored order" step and
	// its thenKeys resync: the nested-Then loop right below reads
	// res.items[sub.parentSet]/res.thenKeys[sub.id], which are
	// items[0]/childKeys/res.thenKeys[...] whenever a nested Then(...) sits
	// directly under this ordered edge (sub.parentSet == 0), so all three
	// must see the same restored order children does — a nested Then(...)
	// that was missed would stitch its parent keys against the wrong row once
	// reordered.
	if ordered {
		var err error
		children, childKeys, err = reorderThenChildren(e, childPlan, &res, children, childKeys)
		if err != nil {
			return nil, nil, err
		}
	}

	// Nested Then boundaries inside the child island.
	for _, sub := range childPlan.thens {
		if err := runThen(ctx, exec, sub, res.items[sub.parentSet], res.thenKeys[sub.id]); err != nil {
			return nil, nil, err
		}
	}

	return children, childKeys, nil
}

// reorderThenChildren applies the edge's requested order to a Then(...)
// island's root rows. children, childKeys and res.thenKeys (for any
// Then(...) nested directly under the island's root) are reassigned
// together; the reordered children and childKeys are returned, while
// res.items[0] and res.thenKeys are updated in place.
func reorderThenChildren(e edgeSpec, childPlan islandPlan, res *islandResult, children, childKeys []any) ([]any, []any, error) {
	// childKeys/res.thenKeys are execIsland's own scan-time output for this
	// island's root (rootChildKeys) and for any Then(...) nested directly
	// under it — never caller-optional here, unlike stitchInto's
	// childKeyVals — so a length mismatch means the key-sync invariant itself
	// broke, not "no keys given", and must fail loudly rather than silently
	// reorder past it.
	if len(childKeys) != len(children) {
		return nil, nil, fmt.Errorf("sectionload: %s: %d then child keys for %d children", e.relName(), len(childKeys), len(children))
	}
	for _, sub := range childPlan.thens {
		if sub.parentSet != 0 {
			continue
		}
		if len(res.thenKeys[sub.id]) != len(children) {
			return nil, nil, fmt.Errorf("sectionload: %s: %d then parent keys for %d children", sub.edge.relName(), len(res.thenKeys[sub.id]), len(children))
		}
	}

	idx := make([]int, len(children))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return res.ords[idx[a]] < res.ords[idx[b]] })
	sorted := make([]any, len(children))
	for i, j := range idx {
		sorted[i] = children[j]
	}
	children = sorted
	res.items[0] = sorted

	sortedKeys, err := reorderAligned(childKeys, idx, e.relName())
	if err != nil {
		return nil, nil, err
	}
	childKeys = sortedKeys

	for _, sub := range childPlan.thens {
		if sub.parentSet != 0 {
			continue
		}
		keys, err := reorderAligned(res.thenKeys[sub.id], idx, sub.edge.relName())
		if err != nil {
			return nil, nil, err
		}
		res.thenKeys[sub.id] = keys
	}
	return children, childKeys, nil
}
