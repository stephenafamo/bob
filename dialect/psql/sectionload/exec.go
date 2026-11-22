package sectionload

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/stephenafamo/bob"
)

// cloneKey defensively copies a key tuple returned by a *KeyFromBuf (or
// ParentKey/ChildKey fallback) callback before it is stored in
// parentKeys/childKeys/thenKeys/rootChildKeys. Nothing in Edge's contract
// requires a callback to return a freshly allocated slice every call;
// scan buffers are reused row to row (see execIsland's own doc below), so
// a callback that returned a slice aliasing buffer state — instead of
// copied values — would otherwise let a later row's scan silently rewrite
// an earlier row's already-stored key. Copying here, once, at the single
// point every key tuple passes through on its way into these maps, closes
// that gap regardless of how a given callback is implemented.
func cloneKey(k []any) []any {
	if k == nil {
		return nil
	}
	out := make([]any, len(k))
	for i, v := range k {
		// mapKey explicitly supports []byte keys; the interface value
		// only carries a slice header, so a shallow copy would still
		// alias the callback's reused backing array.
		if b, ok := v.([]byte); ok {
			out[i] = append([]byte(nil), b...)
			continue
		}
		out[i] = v
	}
	return out
}

// islandResult is everything execIsland extracts from one island's rows.
//
// Named fields rather than positional returns: every value here is either
// indexed by a set number or aligned positionally with another field, and
// mis-indexed positional plumbing is an easy bug to write (thenKeys read
// out of parentKeys[parentSet]; runThen reading parentKeys[0], which
// execIsland never populates). Naming each lane makes that class of
// mistake visible at the call site.
type islandResult struct {
	// items are the built models indexed by set number (0 = the island's
	// own parent/root rows).
	items [][]any
	// parentKeys[s][i] is the join key value(s) that row i of the section
	// s hangs under exposes to section s's own edge. Aligned with
	// items[<that section's parentSet>], not with items[s].
	parentKeys map[int][]any
	// childKeys[s][i] is the join key value(s) items[s][i] exposes as a
	// child of its own edge. Set 0 has none: the root is nobody's child.
	childKeys map[int][]any
	// thenKeys[id][i] is the same idea for a Then(...) boundary, keyed by
	// thenPlan.id — never by parentSet, which collides with an unrelated
	// section's set number.
	thenKeys map[int][]any
	// rootChildKeys[i] is the key items[0][i] exposes as a child of the
	// edge that opened this island. Only runThen populates it (via
	// execIsland's rootChildKeyFn); the top-level All() leaves it nil.
	rootChildKeys []any
	// ords[i] is the ordinality value that restores the parent query's
	// original row order for items[0][i].
	ords []int64
	// secOrds[s][i] is the per-section row ordinal captured for sections
	// whose edge requested an ORDER BY; nil for the rest.
	secOrds [][]sql.NullInt64
	// counts[s][k] is the number of child rows the count section s
	// returned for parent key k. Only the keys the query actually returned
	// are present: giving every OTHER parent a zero is stitchCounts's job,
	// because that is where the full parent list lives.
	counts map[int]map[any]int64
}

// execIsland runs the composed query and dispatches rows by the
// discriminator column into an islandResult.
//
// parentKeys/childKeys/thenKeys/rootChildKeys are all computed from the
// scan buffer at scan time, not from the built model afterward (see
// stitch.go), because the value has to be read while the buffer still
// holds THIS row (a buffer is retired as soon as its row has produced a
// model, but the read still has to happen before that point) and because
// an Internal (join-key) column is never guaranteed to survive into the
// built model. Every value read this way is passed through cloneKey before
// being stored, since a *KeyFromBuf/ParentKey/ChildKey callback is not
// required to return a freshly allocated slice — without the copy, the
// next row's Scan into the same buffer would silently rewrite an
// already-stored key.
//
// rootChildKeyFn, when non-nil, is called for every root (set 0) row to
// produce rootChildKeys: the join key value that row exposes as a child
// of the edge that made this whole island exist in the first place. This
// is only meaningful for an island runThen opened for a Then(...)
// boundary (rootChildKeyFn is e.childKeyFrom there; e.childKeyFrom is
// childKeyFromBufAny for a single-hop e) — the top-level All() call passes
// nil, since its root is the caller's own query, not anyone's child.
// root is read through rowBufs (not bufAny): a Then boundary's own edge
// can itself be multi-hop, in which case root's row also has a carried
// buffer (rootCarriedBuf below) childKeyFrom needs to read the value from.
func execIsland(
	ctx context.Context,
	exec bob.Executor,
	q bob.Query,
	root partsSpec,
	plan islandPlan,
	rootChildKeyFn func(rowBufs) []any,
) (islandResult, error) {
	sqlStr, args, err := bob.Build(ctx, q)
	if err != nil {
		return islandResult{}, fmt.Errorf("sectionload: building query: %w", err)
	}

	rows, err := exec.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return islandResult{}, fmt.Errorf("sectionload: executing query: %w", err)
	}
	defer rows.Close()

	// After the query succeeds, verify the result's column layout by position,
	// so a shifted scan cannot go unnoticed. The database may reject a parent
	// column subset itself before execution ever reaches this point.
	got, err := rows.Columns()
	if err != nil {
		return islandResult{}, err
	}
	if err := checkResultColumns(got, root, plan); err != nil {
		return islandResult{}, err
	}

	s := newIslandScan(root, plan, rootChildKeyFn)

	for rows.Next() {
		if err := rows.Scan(s.targets...); err != nil {
			return islandResult{}, fmt.Errorf("sectionload: scanning row: %w", err)
		}
		if err := s.consumeRow(); err != nil {
			return islandResult{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return islandResult{}, err
	}
	return s.res, nil
}

// checkResultColumns compares the result set's column names (got) with the
// layout expectedColumns derives from root and plan.
func checkResultColumns(got []string, root partsSpec, plan islandPlan) error {
	expected := expectedColumns(root, plan)
	if len(got) != len(expected) {
		return &LayoutError{Expected: expected, Got: got}
	}
	for i := range got {
		if got[i] != expected[i] {
			return &LayoutError{Expected: expected, Got: got, FirstDiff: i}
		}
	}
	return nil
}

// islandScan holds everything execIsland needs while it walks the rows of
// one island: the scan target slice, the buffers the targets point into,
// the offsets that say which slots belong to whom, and the result being
// built.
//
// The target SLICE is assembled once: its length and the column each slot
// maps to are fixed for the whole result set. The BUFFERS it points into
// are not fixed — a buffer whose row has produced a model is retired right
// there and a fresh one takes over its slots (see refreshBuf).
// rootOff/rootLen and secOff/secLen record which slots belong to whom so
// that swap can rewrite exactly those.
type islandScan struct {
	root           partsSpec
	plan           islandPlan
	rootChildKeyFn func(rowBufs) []any

	set     int64
	ord     sql.NullInt64
	targets []any

	rootBuf bufAny
	rootOff int
	rootLen int

	// rootCarriedBuf/rootCarriedOff/rootCarriedLen: only non-nil
	// when root is an edgeRootParts wrapping a Then(...) boundary whose
	// OWN edge is multi-hop (hooks.go's runThen; the true top-level root
	// from run.go always returns nil here). Scanned as a SEPARATE target
	// range, right after the root's own — never merged into rootBuf's
	// range — because the carried value's Go type is Buf[P] (the
	// GRANDPARENT's own type, one level further up than root itself),
	// not root's own row type; the two buffers cannot be the same object.
	rootCarriedBuf bufAny
	rootCarriedOff int
	rootCarriedLen int

	declaredCountCells []*sql.NullInt64

	bufs   map[int]bufAny
	bySet  map[int]flatNode
	secOff map[int]int
	secLen map[int]int
	// carriedBufs/carriedOff/carriedLen: the SAME shape as
	// bufs/secOff/secLen, one entry per section whose edge is multi-hop
	// (absent — not merely nil — for a single-hop section, so a lookup
	// miss and "no carried buffer" are the same thing). A separate
	// map, not a second range tacked onto secOff/secLen, because the two
	// buffers are refreshed independently and have
	// different Go types (Buf[C] vs Buf[P]).
	carriedBufs map[int]bufAny
	carriedOff  map[int]int
	carriedLen  map[int]int

	countBufs  map[int]bufAny
	countCells map[int]*sql.NullInt64
	countBySet map[int]countPlan
	childrenOf map[int][]flatNode
	thensOf    map[int][]thenPlan
	countsOf   map[int][]countPlan
	res        islandResult
}

// newIslandScan assembles the scan targets, per-section buffers, lookup
// maps and the empty result for one island.
func newIslandScan(root partsSpec, plan islandPlan, rootChildKeyFn func(rowBufs) []any) *islandScan {
	s := &islandScan{root: root, plan: plan, rootChildKeyFn: rootChildKeyFn}
	s.targets = []any{&s.set, &s.ord}

	s.addRootTargets()
	s.addDeclaredCountTargets()
	s.addSectionTargets()
	s.addCountTargets()
	s.buildParentIndexes()
	s.initResult()
	return s
}

// addRootTargets appends the root's own scan targets and, when present, the
// root's carried buffer targets.
//
// root.scanCols(), NOT root.cols(): root.newBufAny() is the ROW type's own
// buffer (for the true top-level parent, run.go's partsAdapter; for a
// Then(...) boundary's child, edgeRootParts delegating to e.child()'s
// partsAdapter) — it has no field for a carried column, which belongs
// to an entirely different table (the parent's, one level further up
// — see parts.go's Edge.Carried doc). Passing root.cols() (scan+
// carried) here would ask that buffer to scan a column it does not
// have: a real generated Buf.TargetsFor panics on the unrecognized
// name (parts.go's own documented contract). root.cols()
// remains correct wherever it names the FULL layout width
// (expectedColumns, section0List, checkWidth) — only the actual scan
// target assembly changes here. For the true top-level root
// (partsAdapter), scanCols() == cols() always, so this is a no-op
// there.
func (s *islandScan) addRootTargets() {
	s.rootBuf = s.root.newBufAny()
	s.rootOff = len(s.targets)
	s.targets = append(s.targets, s.rootBuf.targetsFor(s.root.scanCols())...)
	s.rootLen = len(s.targets) - s.rootOff

	if cb := s.root.newCarriedBufAny(); cb != nil {
		s.rootCarriedBuf = cb
		s.rootCarriedOff = len(s.targets)
		s.targets = append(s.targets, cb.targetsFor(s.root.carriedScanCols())...)
		s.rootCarriedLen = len(s.targets) - s.rootCarriedOff
	}
}

// addDeclaredCountTargets adds extra scalar columns for the root only.
// root.cols() = scanCols ++ carriedCols, so the declared count columns
// must come after the carried ones or the order would not match
// expectedColumns. This follows the same throwaway-cell pattern as
// countCells.
func (s *islandScan) addDeclaredCountTargets() {
	s.declaredCountCells = make([]*sql.NullInt64, len(s.plan.declaredCounts))
	for i := range s.plan.declaredCounts {
		cell := new(sql.NullInt64)
		s.declaredCountCells[i] = cell
		s.targets = append(s.targets, cell)
	}
}

// addSectionTargets appends, per row section, the child buffer's targets
// and, for a multi-hop edge, the carried buffer's targets.
func (s *islandScan) addSectionTargets() {
	n := len(s.plan.sections)
	s.bufs = make(map[int]bufAny, n)
	s.bySet = make(map[int]flatNode, n)
	s.secOff = make(map[int]int, n)
	s.secLen = make(map[int]int, n)
	s.carriedBufs = make(map[int]bufAny, n)
	s.carriedOff = make(map[int]int, n)
	s.carriedLen = make(map[int]int, n)
	for _, sec := range s.plan.sections {
		b := sec.edge.child().newBufAny()
		s.bufs[sec.set] = b
		s.bySet[sec.set] = sec
		off := len(s.targets)
		// sec.edge.childScanCols() (NOT childCols(), and NOT
		// sec.edge.child().cols()): the model buffer only ever knows how to
		// scan the CHILD table's own columns. A multi-hop edge's carried
		// column(s) are a DIFFERENT table's value (the parent's own join
		// key, read back via a JOIN — parts.go's Edge.Carried doc) and go
		// into the separate carried buffer below, never into b.
		s.targets = append(s.targets, b.targetsFor(sec.edge.childScanCols())...)
		s.secOff[sec.set] = off
		s.secLen[sec.set] = len(s.targets) - off

		if cb := sec.edge.newCarriedBufAny(); cb != nil {
			s.carriedBufs[sec.set] = cb
			coff := len(s.targets)
			s.targets = append(s.targets, cb.targetsFor(sec.edge.carriedScanCols())...)
			s.carriedOff[sec.set] = coff
			s.carriedLen[sec.set] = len(s.targets) - coff
		}
	}
}

// addCountTargets appends the count sections, which occupy the tail of the
// layout: their join key columns followed by their own _sl_cnt<set> column.
// The count column is NULL-padded in every other branch, so it is scanned
// through a sql.NullInt64 cell rather than a plain int64.
func (s *islandScan) addCountTargets() {
	s.countBufs = make(map[int]bufAny, len(s.plan.counts))
	s.countCells = make(map[int]*sql.NullInt64, len(s.plan.counts))
	s.countBySet = make(map[int]countPlan, len(s.plan.counts))
	for _, c := range s.plan.counts {
		var b bufAny
		var scanCols []Column
		if len(c.edge.sides()) > 1 {
			// For a multi-hop count, keyCols names real columns of the first
			// intermediate table, but those names do not exist in the FINAL child
			// table's Buf[C]. Read the PARENT's Buf[P] by the PARENT's own column
			// names (carriedScanCols) instead.
			b = c.edge.newCarriedBufAny()
			scanCols = c.edge.carriedScanCols()
		} else {
			b = c.edge.child().newBufAny()
			scanCols = c.keyCols
		}
		s.countBufs[c.set] = b
		s.countBySet[c.set] = c
		s.targets = append(s.targets, b.targetsFor(scanCols)...)
		cell := new(sql.NullInt64)
		s.countCells[c.set] = cell
		s.targets = append(s.targets, cell)
	}
}

// buildParentIndexes builds the by-parent-set indexes.
//
// childrenOf[s] = sections whose parentSet is s, i.e. the edges that
// will need a parent-key reading from a row belonging to set s.
//
// thensOf[s] = Then(...) boundaries whose parentSet is s — the same
// idea as childrenOf, for the id-keyed thenKeys map instead.
//
// countsOf[s] = count sections hanging under set s, i.e. the count
// edges that need a parent-key reading from a row belonging to set s.
func (s *islandScan) buildParentIndexes() {
	s.childrenOf = make(map[int][]flatNode, len(s.plan.sections)+1)
	for _, n := range s.plan.sections {
		s.childrenOf[n.parentSet] = append(s.childrenOf[n.parentSet], n)
	}

	s.thensOf = make(map[int][]thenPlan, len(s.plan.thens))
	for _, tp := range s.plan.thens {
		s.thensOf[tp.parentSet] = append(s.thensOf[tp.parentSet], tp)
	}

	s.countsOf = make(map[int][]countPlan, len(s.plan.counts))
	for _, c := range s.plan.counts {
		s.countsOf[c.parentSet] = append(s.countsOf[c.parentSet], c)
	}
}

// initResult allocates the empty islandResult.
func (s *islandScan) initResult() {
	slots := resultSetSlots(s.plan)
	s.res.items = make([][]any, slots)
	s.res.parentKeys = make(map[int][]any, len(s.plan.sections)+1)
	s.res.childKeys = make(map[int][]any, len(s.plan.sections))
	s.res.thenKeys = make(map[int][]any, len(s.plan.thens))
	s.res.secOrds = make([][]sql.NullInt64, slots)
	s.res.counts = make(map[int]map[any]int64, len(s.plan.counts))
	for _, c := range s.plan.counts {
		s.res.counts[c.set] = make(map[any]int64)
	}
}

// refreshBuf installs fresh's scan destinations over the [off, off+n)
// slots of targets. It is called right after a row's model has been
// built and its keys extracted, so the NEXT Scan writes into a brand
// new buffer instead of the one the just-built model may still alias.
//
// Why a whole new buffer rather than clearing the old one: a Buf is
// generated code the runtime does not control, and its contract
// (parts.go) is a plain value copy in BuildFor. For a field whose type
// carries a slice — types.JSON[json.RawMessage], whose Scan bottoms out
// in json.RawMessage.UnmarshalJSON's append((*m)[:0], data...), and
// anything else that appends into its own backing array — that copies
// only the slice header. Handing the next row a fresh buffer is the
// only thing the runtime can do on its own to guarantee that row cannot
// reach the previous row's array.
//
// The length check is a defensive invariant, not an expected path:
// TargetsFor is deterministic for a given cols slice, so a fresh buffer
// of the same table always yields the same number of targets.
func (s *islandScan) refreshBuf(fresh bufAny, cols []Column, off, n int) error {
	next := fresh.targetsFor(cols)
	if len(next) != n {
		return fmt.Errorf(
			"sectionload: refreshed scan buffer produced %d targets, want %d",
			len(next), n,
		)
	}
	copy(s.targets[off:off+n], next)
	return nil
}

// consumeRow routes the row just scanned to the root, a row section or a
// count section according to its set discriminator.
func (s *islandScan) consumeRow() error {
	set := int(s.set)
	if set == 0 {
		return s.consumeRootRow()
	}
	if b, ok := s.bufs[set]; ok {
		return s.consumeSectionRow(set, b)
	}
	if cp, ok := s.countBySet[set]; ok {
		return s.consumeCountRow(cp)
	}
	return fmt.Errorf("sectionload: unexpected %s value %d", colSet, s.set)
}

// consumeRootRow handles a set 0 row: build the root model, apply declared
// counts, record the keys the row exposes, and retire the root buffers.
func (s *islandScan) consumeRootRow() error {
	res := &s.res
	res.items[0] = append(res.items[0], s.rootBuf.buildFor(s.root.modelCols()))
	if err := s.applyDeclaredCounts(); err != nil {
		return err
	}
	res.ords = append(res.ords, s.ord.Int64)
	for _, child := range s.childrenOf[0] {
		res.parentKeys[child.set] = append(res.parentKeys[child.set], cloneKey(child.edge.parentKeyFromBufAny(s.rootBuf)))
	}
	for _, tp := range s.thensOf[0] {
		res.thenKeys[tp.id] = append(res.thenKeys[tp.id], cloneKey(tp.edge.parentKeyFromBufAny(s.rootBuf)))
	}
	for _, cp := range s.countsOf[0] {
		res.parentKeys[cp.set] = append(res.parentKeys[cp.set], cloneKey(cp.edge.parentKeyFromBufAny(s.rootBuf)))
	}
	if s.rootChildKeyFn != nil {
		res.rootChildKeys = append(res.rootChildKeys, cloneKey(s.rootChildKeyFn(rowBufs{model: s.rootBuf, carried: s.rootCarriedBuf})))
	}
	// Every read of this row's root buffer is done by here (model
	// built, every key extracted and cloned), so retire it: the next
	// Scan must not be able to write through the model just appended
	// to res.items[0].
	s.rootBuf = s.root.newBufAny()
	// root.scanCols(), not root.cols() — must match what was used
	// at setup time, or the replacement buffer's target
	// count differs from rootLen (same as childScanCols() for a
	// section's own buffer).
	if err := s.refreshBuf(s.rootBuf, s.root.scanCols(), s.rootOff, s.rootLen); err != nil {
		return err
	}
	// The root's carried buffer (only non-nil for a Then
	// boundary whose own edge is multi-hop) is retired the same
	// way, right after rootChildKeyFn above has read it.
	if s.rootCarriedBuf != nil {
		fresh := s.root.newCarriedBufAny()
		if err := s.refreshBuf(fresh, s.root.carriedScanCols(), s.rootCarriedOff, s.rootCarriedLen); err != nil {
			return err
		}
		s.rootCarriedBuf = fresh
	}
	return nil
}

// applyDeclaredCounts applies the declared count preloads to the root
// model just appended. The declared name/type cannot skew within a single
// code generation run, so the assertion should always hold. A failure is
// reported as an explicit error rather than silently skipped: a silent
// no-op would hide a broken assumption and make the reasoning behind the
// assertion moot.
func (s *islandScan) applyDeclaredCounts() error {
	if len(s.plan.declaredCounts) == 0 {
		return nil
	}
	built := s.res.items[0][len(s.res.items[0])-1]
	loader, ok := built.(countPreloadable)
	if !ok {
		return fmt.Errorf(
			"sectionload: %T does not implement countPreloadable, but a PreloadCount was declared on the parent query",
			built,
		)
	}
	for i, dc := range s.plan.declaredCounts {
		cell := s.declaredCountCells[i]
		if !cell.Valid {
			continue // a NULL on a parent row is not expected; skip to be safe (matches bob's native PreloadCount, which also does not call the hook when the correlated subquery is NULL)
		}
		if err := loader.PreloadCount(dc.RelName, cell.Int64); err != nil {
			return fmt.Errorf("sectionload: %s: %w", dc.RelName, err)
		}
	}
	return nil
}

// consumeSectionRow handles a row of a row section whose buffer is b.
func (s *islandScan) consumeSectionRow(set int, b bufAny) error {
	res := &s.res
	n := s.bySet[set]
	res.items[set] = append(res.items[set], b.buildFor(n.edge.childModelCols()))
	res.childKeys[set] = append(res.childKeys[set], cloneKey(n.edge.childKeyFrom(rowBufs{model: b, carried: s.carriedBufs[set]})))
	res.secOrds[set] = append(res.secOrds[set], s.ord)
	for _, child := range s.childrenOf[set] {
		res.parentKeys[child.set] = append(res.parentKeys[child.set], cloneKey(child.edge.parentKeyFromBufAny(b)))
	}
	for _, tp := range s.thensOf[set] {
		res.thenKeys[tp.id] = append(res.thenKeys[tp.id], cloneKey(tp.edge.parentKeyFromBufAny(b)))
	}
	for _, cp := range s.countsOf[set] {
		res.parentKeys[cp.set] = append(res.parentKeys[cp.set], cloneKey(cp.edge.parentKeyFromBufAny(b)))
	}
	// Same as the root branch: this section's buffer has been fully
	// read, so retire it before the next Scan. Only
	// THIS section's buffer is replaced — every other section's
	// buffer merely took NULL padding on this row and still holds
	// nothing anyone kept.
	nb := n.edge.child().newBufAny()
	// n.edge.childScanCols() (NOT childCols(), and NOT
	// n.edge.child().cols()) must match what was used at setup time
	// or the replacement buffer's target count will differ — b only
	// ever scanned the child table's own columns; the carried
	// buffer (if any) is refreshed separately right below.
	if err := s.refreshBuf(nb, n.edge.childScanCols(), s.secOff[set], s.secLen[set]); err != nil {
		return err
	}
	s.bufs[set] = nb
	if _, ok := s.carriedBufs[set]; ok {
		fresh := n.edge.newCarriedBufAny()
		if err := s.refreshBuf(fresh, n.edge.carriedScanCols(), s.carriedOff[set], s.carriedLen[set]); err != nil {
			return err
		}
		s.carriedBufs[set] = fresh
	}
	return nil
}

// consumeCountRow handles a row of the count section cp.
//
// A count row builds no model: only its grouped key and its
// count(*) matter. A NULL key group (children whose nullable FK
// is not set) matches no parent, so it is dropped here rather
// than carried into the key map. mapKey normalizes (and, for
// []byte, copies) kv immediately below, and the result is used
// only as a map key within this same call — it never
// survives to the next row's Scan, so cloneKey is unnecessary
// here (unlike the key slices above, which are appended to
// slices that persist across every remaining row). For the
// same reason a count buffer is never retired the way a root
// or section buffer is: a count row builds no
// model, and nothing that outlives this call points into
// that buffer.
func (s *islandScan) consumeCountRow(cp countPlan) error {
	kv := cp.edge.childKeyFrom(s.countBufs[cp.set])
	if kv == nil {
		return nil
	}
	k, err := mapKey(kv)
	if err != nil {
		return fmt.Errorf("sectionload: %s count key: %w", cp.edge.relName(), err)
	}
	s.res.counts[cp.set][k] = s.countCells[cp.set].Int64
	return nil
}

// resultSetSlots returns the number of slots res.items/res.secOrds need:
// one past the highest set number in use across row sections AND count
// sections. planIsland draws both from the same counter (interleaved in
// declaration order), so len(plan.sections)+1 undercounts whenever a count
// node is declared before a row section — the row section's set can then
// exceed len(plan.sections), and indexing res.items[set] would panic with
// index out of range.
func resultSetSlots(plan islandPlan) int {
	maxSet := 0
	for _, n := range plan.sections {
		if n.set > maxSet {
			maxSet = n.set
		}
	}
	for _, c := range plan.counts {
		if c.set > maxSet {
			maxSet = c.set
		}
	}
	return maxSet + 1
}

func expectedColumns(root partsSpec, plan islandPlan) []string {
	out := []string{colSet, colOrd}
	for _, c := range root.cols() {
		out = append(out, c.Name)
	}
	// Right after root.cols() (scanCols ++ carriedCols), list the declared
	// count preload aliases. The scan target assembly in execIsland (the
	// root's target additions) must follow the same order.
	for _, dc := range plan.declaredCounts {
		out = append(out, dc.Alias)
	}
	for _, n := range plan.sections {
		for _, c := range n.edge.childCols() {
			out = append(out, c.Name)
		}
	}
	for _, cp := range plan.counts {
		for _, c := range cp.cols() {
			out = append(out, c.Name)
		}
	}
	return out
}
