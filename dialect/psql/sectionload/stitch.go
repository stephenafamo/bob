package sectionload

import (
	"database/sql"
	"fmt"
	"sort"
)

// stitchIsland connects children to parents, deepest nodes first, so
// that R fields are populated bottom-up. All R.<Rel> / R.Loaded /
// back-reference semantics live in the generated Stitch functions
// (which call the models' Preload methods).
//
// It reads res.parentKeys/res.childKeys — execIsland's precomputed,
// buffer-derived join key values: res.parentKeys[n.set] is aligned with
// res.items[n.parentSet] (not res.items[n.set] — it holds each PARENT
// row's key as seen by THIS edge) and res.childKeys[n.set] is aligned
// with res.items[n.set] itself. Reading keys from the model here would
// break once a join-key column is excluded from the model (an
// Internal-only column) — see stitchInto's doc comment.
// res.secOrds carries, per section, the row-ordinal column captured for
// sections whose edge requested an ORDER BY; it is nil-safe (a nil/empty
// slice for a section disables sorting for that edge).
//
// stitchInto's sort result is written back into res.items[n.set]: a
// nested Then(...) sitting directly under this edge reads
// res.items[tp.parentSet] (== res.items[n.set] here) afterward, and must
// see the same order stitching used — the same requirement as for a Then
// boundary's own root. plan.thens/res.thenKeys extend that to
// res.thenKeys[tp.id] for every Then(...) boundary nested directly under
// this section (tp.parentSet == n.set): res.thenKeys was built by
// execIsland in scan order, so once stitchInto reorders this section's
// own items by secOrds, thenKeys for anything hanging off those items
// must be carried along with the exact same permutation, or a boundary
// nested under an ordered section stitches its parent keys against the
// wrong row (the same bug class as runThen's own childKeys/thenKeysBySet
// resync, but for a same-island section instead of a Then boundary's own
// root).
//
// Count sections need none of this reordering — a count has no row order
// to preserve — so they are stitched last, in one pass, via stitchCounts.
func stitchIsland(plan islandPlan, res islandResult) error {
	thensBySet := make(map[int][]thenPlan, len(plan.thens))
	for _, tp := range plan.thens {
		thensBySet[tp.parentSet] = append(thensBySet[tp.parentSet], tp)
	}

	for i := len(plan.sections) - 1; i >= 0; i-- {
		n := plan.sections[i]
		var ords []sql.NullInt64
		if res.secOrds != nil {
			ords = res.secOrds[n.set]
		}
		sorted, err := stitchInto(n.edge, res.items[n.parentSet], res.items[n.set], res.parentKeys[n.set], res.childKeys[n.set], ords)
		if err != nil {
			return err
		}
		res.items[n.set] = sorted

		idx, err := sortIndexByOrds(ords, len(sorted), n.edge.relName())
		if err != nil {
			return err
		}
		for _, tp := range thensBySet[n.set] {
			keys := res.thenKeys[tp.id]
			// res.thenKeys[tp.id] is execIsland's own scan-time output (part of
			// the key-sync invariant), never a caller-optional value the way
			// stitchInto's childKeyVals is — a length mismatch here means the sync
			// invariant itself broke, not "no keys given", so this must always be
			// checked, ORDER BY or not: silently skipping the check whenever idx was
			// nil would hide exactly this class of bug.
			if len(keys) != len(sorted) {
				return fmt.Errorf("sectionload: %s: %d then parent keys for %d parents", tp.edge.relName(), len(keys), len(sorted))
			}
			if idx != nil {
				keys, err = reorderAligned(keys, idx, tp.edge.relName())
				if err != nil {
					return err
				}
			}
			res.thenKeys[tp.id] = keys
		}
	}
	return stitchCounts(plan.counts, res.items, res.parentKeys, res.counts)
}

// stitchCounts stores one count on EVERY parent of every count section,
// including the parents no count row came back for.
//
// This is where counts differ from rows on purpose. stitchInto leaves a
// parent untouched when its (nullable) join key is not set; for a to-one
// edge, it also leaves the parent untouched when no child is found. bob's
// own LoadCount<Rel> assigns a count to every parent instead: it builds a
// key→count map from the aggregate rows and then assigns &count to every
// parent unconditionally, so a parent with no matching children gets a
// pointer to 0 (the generated counts template does
// `count := countMap[o.<PK>]`, which on a missing key is Go's zero
// value). Reproducing that is the whole point of a count node: nil means
// "not loaded", &0 means "loaded, zero children".
//
// parentKeys[c.set] is execIsland's buffer-derived key per parent row,
// positionally aligned with items[c.parentSet]; the model-based fallback
// mirrors stitchInto's and only applies when no keys were precomputed.
func stitchCounts(counts []countPlan, items [][]any, parentKeys map[int][]any, countVals map[int]map[any]int64) error {
	for _, c := range counts {
		parents := items[c.parentSet]
		keys := parentKeys[c.set]
		if keys != nil && len(keys) != len(parents) {
			return fmt.Errorf("sectionload: %s: %d parent keys for %d parents", c.edge.relName(), len(keys), len(parents))
		}
		byKey := countVals[c.set]
		for i, p := range parents {
			var kv []any
			if keys != nil {
				kv, _ = keys[i].([]any)
			} else {
				kv = c.edge.parentKeyAny(p)
			}
			var n int64
			if kv != nil {
				k, err := mapKey(kv)
				if err != nil {
					return fmt.Errorf("sectionload: %s parent key: %w", c.edge.relName(), err)
				}
				n = byKey[k]
			}
			if err := c.edge.setCountAny(p, n); err != nil {
				return fmt.Errorf("sectionload: setting %s count: %w", c.edge.relName(), err)
			}
		}
	}
	return nil
}

// sortIndexByOrds returns the permutation that stable-sorts n items by
// ords ascending (the same rule stitchInto applies to children), or nil
// when no ORDER BY was requested for this section (len(ords) == 0, or
// every entry is an invalid sql.NullInt64 — the same "no ordinal captured"
// state, just spelled two ways depending on the caller). Shared by
// stitchInto and stitchIsland so both agree, by construction, on which row
// moved where.
//
// len(ords) not matching n, or only some (not all/none) of ords being
// valid, is a caller bug — execIsland always produces exactly one ords
// entry per row, all valid or all invalid together for a given section —
// and is reported as an error rather than silently treated as "no ORDER
// BY": treating a length mismatch as "no ORDER BY" would conflate "no
// ORDER BY requested" with "ords and the row set have drifted apart", so
// a future change that broke the sync invariant would silently fall back
// to unordered/model-key behavior instead of failing loudly.
func sortIndexByOrds(ords []sql.NullInt64, n int, relName string) ([]int, error) {
	if len(ords) == 0 {
		return nil, nil
	}
	if len(ords) != n {
		return nil, fmt.Errorf("sectionload: %s: %d ordinal positions for %d values", relName, len(ords), n)
	}

	valid := 0
	for _, ord := range ords {
		if ord.Valid {
			valid++
		}
	}
	if valid == 0 {
		return nil, nil
	}
	if valid != n {
		return nil, fmt.Errorf("sectionload: %s: %d valid ordinal positions for %d values", relName, valid, n)
	}

	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return ords[idx[a]].Int64 < ords[idx[b]].Int64 })
	return idx, nil
}

// reorderAligned applies idx to vals: out[i] = vals[idx[i]]. Returns vals
// unchanged when idx is nil (no reordering requested). relName labels the
// error when vals's length disagrees with idx's — either a caller bug (idx
// was built from the same row set vals came from) or, for a nil vals a
// caller passed deliberately (e.g. stitchInto's childKeyVals, which may be
// omitted to fall back to model-derived keys), a caller that forgot idx
// can be non-nil in that case too and must guard the call itself rather
// than rely on this function to no-op (an early return on `vals == nil`
// would make a deliberately omitted childKeyVals and a same-invariant-
// violation nil indistinguishable from each other).
func reorderAligned(vals []any, idx []int, relName string) ([]any, error) {
	if idx == nil {
		return vals, nil
	}
	if len(vals) != len(idx) {
		return nil, fmt.Errorf("sectionload: %s: %d values for %d ordinal positions", relName, len(vals), len(idx))
	}
	out := make([]any, len(vals))
	for i, j := range idx {
		out[i] = vals[j]
	}
	return out, nil
}

// stitchInto links children into parents through one edge, and returns
// children in the order actually used for stitching (see stitchIsland's
// doc comment on why the caller must write this back). Used both for
// in-island edges and for Then boundaries.
//
// parentKeyVals[i]/childKeyVals[i] are the pre-extracted join key values
// for parents[i]/children[i], computed from each row's scan buffer at
// scan time (execIsland), not derived from the model here. This is load
// bearing, not a style choice: an Internal join-key column is scanned
// into the buffer but is never guaranteed to be copied into the model,
// and even when a key column IS on the model, the scan buffer backing it
// is reused across rows so it cannot be re-read after the fact. Both
// slices must be nil (skip precomputed keys, e.g. from a caller that
// hasn't been generated to compute them yet) or exactly
// len(parents)/len(children) — anything else is a caller bug.
//
// ords, when non-empty, is the UNION ALL branch's captured row-ordinal for
// each entry of children, in the same order — present only when the edge's
// own SelectOptions requested an ORDER BY: a UNION ALL does not preserve
// any one branch's internal row order, so without this, an edge-level
// ORDER BY would have no visible effect on the resulting R.<Rel> slice.
// children (and childKeyVals, kept aligned with it) are stable-sorted by
// this ordinal, once, before grouping by parent key, so relative order
// survives the group-by exactly as it would for bob's own ThenLoad (a
// plain ordered SELECT, never unioned). When ords is empty (no ORDER BY
// requested), this is a no-op: children keep whatever order the query
// happened to return them in.
func stitchInto(e edgeSpec, parents, children, parentKeyVals, childKeyVals []any, ords []sql.NullInt64) ([]any, error) {
	if parentKeyVals != nil && len(parentKeyVals) != len(parents) {
		return nil, fmt.Errorf("sectionload: %s: %d parent keys for %d parents", e.relName(), len(parentKeyVals), len(parents))
	}
	if childKeyVals != nil && len(childKeyVals) != len(children) {
		return nil, fmt.Errorf("sectionload: %s: %d child keys for %d children", e.relName(), len(childKeyVals), len(children))
	}

	idx, err := sortIndexByOrds(ords, len(children), e.relName())
	if err != nil {
		return nil, err
	}
	if idx != nil {
		sortedChildren := make([]any, len(children))
		for i, j := range idx {
			sortedChildren[i] = children[j]
		}
		children = sortedChildren

		// childKeyVals == nil is stitchInto's own documented opt-out (a caller
		// not yet generated to precompute keys falls back to e.childKeyAny(c)
		// below) — reorderAligned's nil check does not cover this case, so it
		// must be guarded here instead.
		if childKeyVals != nil {
			childKeyVals, err = reorderAligned(childKeyVals, idx, e.relName())
			if err != nil {
				return nil, err
			}
		}
	}

	// Fail closed: childKeyVals == nil is stitchInto's documented opt-out,
	// falling back to e.childKeyAny(c) — the child MODEL's own ChildKey. That
	// fallback is safe for a single-hop edge (the model has the parent's join
	// key column), but for a multi-hop edge it would silently read the FINAL
	// table's ToCols instead of the parent's actual key, stitching every
	// child onto the wrong group with no error at all — the premise of a
	// multi-hop edge is that the final table has no column holding the real
	// key. execIsland always supplies childKeyVals for every edge it scans
	// (sections via n.edge.childKeyFrom, Then-boundary roots via
	// rootChildKeyFn), so a nil childKeyVals for a multi-hop edge here means
	// some caller bypassed execIsland entirely — reject it rather than stitch
	// with silently wrong keys.
	if childKeyVals == nil && len(e.sides()) > 1 {
		return nil, &CarriedKeyMismatchError{
			RelName: e.relName(),
			Reason:  "no precomputed child key was supplied for a multi-hop edge; the model-based ChildKey fallback would read the wrong (final-table) columns",
		}
	}

	groups := make(map[any][]any, len(children))
	for i, c := range children {
		var kv any
		if childKeyVals != nil {
			kv = childKeyVals[i]
		} else {
			kv = e.childKeyAny(c)
		}
		k, err := mapKey(kv.([]any))
		if err != nil {
			return nil, fmt.Errorf("sectionload: %s child key: %w", e.relName(), err)
		}
		groups[k] = append(groups[k], c)
	}

	for i, p := range parents {
		var kv []any
		if parentKeyVals != nil {
			kv, _ = parentKeyVals[i].([]any)
		} else {
			kv = e.parentKeyAny(p)
		}
		if kv == nil {
			// Nullable FK not set: no link possible. to-one still stitches (typed
			// nil, R.Loaded=true — bob's own Preload/ThenLoad always sets Loaded
			// regardless of whether a child was found). to-many skips entirely here,
			// leaving the parent untouched: a to-many edge can only reach this
			// branch if it were generated with a nullable parent key, which a to-many
			// edge normally never has. Calling Stitch unconditionally for every
			// parent would silently change to-many's contract.
			if e.isToMany() {
				continue
			}
			if err := e.stitchAny(p, nil); err != nil {
				return nil, fmt.Errorf("sectionload: stitching %s: %w", e.relName(), err)
			}
			continue
		}
		k, err := mapKey(kv)
		if err != nil {
			return nil, fmt.Errorf("sectionload: %s parent key: %w", e.relName(), err)
		}
		// groups[k] is nil when there's no match — fine for both shapes: a
		// to-one edge stitches (typed nil, Loaded=true); a to-many edge stitches
		// an empty slice.
		if err := e.stitchAny(p, groups[k]); err != nil {
			return nil, fmt.Errorf("sectionload: stitching %s: %w", e.relName(), err)
		}
	}
	return children, nil
}

// mapKey normalizes extracted key values into a comparable map key.
// []byte is converted to string; composite keys are packed into small
// fixed-size arrays (comparable as long as the elements are).
//
// vals is never modified. The normalized values go straight into the
// returned key, so a caller may safely retain or reuse the slice it
// passed in. Two current callers depend on that: runThen (hooks.go) keeps
// the very slice it passed here and later hands it to buildEdgeSelect as
// a Then-boundary match value — i.e. as an SQL parameter, where a []byte
// silently turned into a string would be the wrong type — and
// stitchCounts passes elements of the persistent parentKeys map rather
// than a throwaway per-row slice.
func mapKey(vals []any) (any, error) {
	switch len(vals) {
	case 1:
		return normKeyVal(vals[0]), nil
	case 2:
		return [2]any{normKeyVal(vals[0]), normKeyVal(vals[1])}, nil
	case 3:
		a := [3]any(vals) // len(vals) == 3 here, so the conversion cannot panic
		return [3]any{normKeyVal(a[0]), normKeyVal(a[1]), normKeyVal(a[2])}, nil
	case 4:
		a := [4]any(vals) // len(vals) == 4 here, so the conversion cannot panic
		return [4]any{normKeyVal(a[0]), normKeyVal(a[1]), normKeyVal(a[2]), normKeyVal(a[3])}, nil
	default:
		return nil, fmt.Errorf("unsupported key width %d (max 4)", len(vals))
	}
}

// normKeyVal converts a []byte key element into a comparable string; every
// other value passes through unchanged. Returning the converted value
// instead of writing it back into the caller's slice is what keeps mapKey
// free of side effects.
func normKeyVal(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}
