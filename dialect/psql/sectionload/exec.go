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
// requires a callback to return a freshly allocated slice every call (P-03
// Minor, Codex review 2026-08-09); scan buffers are reused row to row (see
// execIsland's own doc below), so a callback that returned a slice
// aliasing buffer state — instead of copied values — would otherwise let a
// later row's scan silently rewrite an earlier row's already-stored key.
// Copying here, once, at the single point every key tuple passes through
// on its way into these maps, closes that gap regardless of how a given
// callback is implemented.
func cloneKey(k []any) []any {
	if k == nil {
		return nil
	}
	out := make([]any, len(k))
	for i, v := range k {
		// mapKey explicitly supports []byte keys; the interface value
		// only carries a slice header, so a shallow copy would still
		// alias the callback's reused backing array (P-03 round 2
		// Important 4).
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
// BOTH bugs issue P-28 found were mis-indexed positional plumbing
// (thenKeys read out of parentKeys[parentSet]; runThen reading
// parentKeys[0], which execIsland never populates). Naming each lane makes
// that class of mistake visible at the call site.
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
	// section's set number (issue P-28).
	thenKeys map[int][]any
	// rootChildKeys[i] is the key items[0][i] exposes as a child of the
	// edge that opened this island. Only runThen populates it (via
	// execIsland's rootChildKeyFn); the top-level All() leaves it nil.
	rootChildKeys []any
	// ords[i] is the ordinality value that restores the parent query's
	// original row order for items[0][i].
	ords []int64
	// secOrds[s][i] is the per-section row ordinal captured for sections
	// whose edge requested an ORDER BY (issue P-02); nil for the rest.
	secOrds [][]sql.NullInt64
	// counts[s][k] is the number of child rows the count section s
	// returned for parent key k. Only the keys the query actually returned
	// are present: giving every OTHER parent a zero is stitchCounts's job,
	// because that is where the full parent list lives (issue P-04 §10.3
	// items 5-6).
	counts map[int]map[any]int64
}

// execIsland runs the composed query and dispatches rows by the
// discriminator column into an islandResult.
//
// parentKeys/childKeys/thenKeys/rootChildKeys are all computed from the
// scan buffer at scan time, not from the built model afterward (see
// stitch.go), because the value has to be read while the buffer still
// holds THIS row (P-03 §11.4改訂 訂正3 — issue P-26 now retires a buffer
// as soon as its row has produced a model, but the read still has to
// happen before that point) and because a P-03 Internal (join-key) column
// is never guaranteed to survive into the built model. Every value read
// this way is passed through cloneKey before being stored, since a
// *KeyFromBuf/ParentKey/ChildKey callback is not required to return a freshly
// allocated slice (P-03 Minor, Codex review 2026-08-09) — without the
// copy, the next row's Scan into the same buffer would silently rewrite an
// already-stored key.
//
// rootChildKeyFn, when non-nil, is called for every root (set 0) row to
// produce rootChildKeys: the join key value that row exposes as a child
// of the edge that made this whole island exist in the first place. This
// is only meaningful for an island runThen opened for a Then(...)
// boundary (rootChildKeyFn is e.childKeyFrom there, P-05 — e.childKeyFrom
// is childKeyFromBufAny for a single-hop e, so this is unchanged for
// every pre-P-05 caller) — the top-level All() call passes nil, since its
// root is the caller's own query, not anyone's child (issue P-28).
// rowBufs (not bufAny), since P-05: a Then boundary's own edge can itself
// be multi-hop, in which case root's row also has a carried buffer
// (rootCarriedBuf below) childKeyFrom needs to read the value from.
func execIsland(
	ctx context.Context,
	exec bob.Executor,
	q bob.Query,
	root partsSpec,
	plan islandPlan,
	rootChildKeyFn func(rowBufs) []any,
) (islandResult, error) {
	var res islandResult

	sqlStr, args, err := bob.Build(ctx, q)
	if err != nil {
		return islandResult{}, fmt.Errorf("sectionload: building query: %w", err)
	}

	rows, err := exec.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return islandResult{}, fmt.Errorf("sectionload: executing query: %w", err)
	}
	defer rows.Close()

	// SQL実行に成功した結果の列配置を位置で検証し、ずれた走査を防ぐ。
	// 親列サブセットは、ここへ到達する前にDBが拒否する場合がある。
	got, err := rows.Columns()
	if err != nil {
		return islandResult{}, err
	}
	expected := expectedColumns(root, plan)
	if len(got) != len(expected) {
		return islandResult{}, &LayoutError{Expected: expected, Got: got}
	}
	for i := range got {
		if got[i] != expected[i] {
			return islandResult{}, &LayoutError{Expected: expected, Got: got, FirstDiff: i}
		}
	}

	// Assemble the scan target SLICE once: its length and the column it
	// maps each slot to are fixed for the whole result set. The BUFFERS it
	// points into are not fixed — a buffer whose row has produced a model
	// is retired right there and a fresh one takes over its slots (issue
	// P-26; see refreshBuf below). rootOff/rootLen and secOff/secLen record
	// which slots belong to whom so that swap can rewrite exactly those.
	var set int64
	var ord sql.NullInt64
	targets := []any{&set, &ord}

	// root.scanCols(), NOT root.cols() (P-05 correction — root.cols() was
	// used here through Task 5's first draft, and is wrong the same way
	// n.edge.childCols() was wrong for a section's own model buffer,
	// Task 5 Step 2/exec.go below: root.newBufAny() is the ROW type's own
	// buffer (for the true top-level parent, run.go's partsAdapter; for a
	// Then(...) boundary's child, edgeRootParts delegating to e.child()'s
	// partsAdapter) — it has no field for a carried column, which belongs
	// to an entirely different table (the parent's, one level further up
	// — see parts.go's Edge.Carried doc). Passing root.cols() (scan+
	// carried) here would ask that buffer to scan a column it does not
	// have: a real generated Buf.TargetsFor panics on the unrecognized
	// name (parts.go's own documented contract), which is exactly why
	// this bug surfaced immediately once Task 8's DB-backed test ran
	// against a real multi-hop Then(...) boundary rather than staying
	// silent behind a permissive hand-written test fixture. root.cols()
	// remains correct wherever it names the FULL layout width
	// (expectedColumns, section0List, checkWidth) — only the actual scan
	// target assembly changes here. For the true top-level root
	// (partsAdapter), scanCols() == cols() always, so this is a no-op
	// there (byte-identical to before P-05, confirmed by TestP05Baseline).
	rootBuf := root.newBufAny()
	rootOff := len(targets)
	targets = append(targets, rootBuf.targetsFor(root.scanCols())...)
	rootLen := len(targets) - rootOff

	// rootCarriedBuf/rootCarriedOff/rootCarriedLen (P-05): only non-nil
	// when root is an edgeRootParts wrapping a Then(...) boundary whose
	// OWN edge is multi-hop (hooks.go's runThen; the true top-level root
	// from run.go always returns nil here). Scanned as a SEPARATE target
	// range, right after the root's own — never merged into rootBuf's
	// range — because the carried value's Go type is Buf[P] (the
	// GRANDPARENT's own type, one level further up than root itself),
	// not root's own row type; the two buffers cannot be the same object.
	var rootCarriedBuf bufAny
	rootCarriedOff, rootCarriedLen := 0, 0
	if cb := root.newCarriedBufAny(); cb != nil {
		rootCarriedBuf = cb
		rootCarriedOff = len(targets)
		targets = append(targets, cb.targetsFor(root.carriedScanCols())...)
		rootCarriedLen = len(targets) - rootCarriedOff
	}

	// declaredCountCells (P-13): root専用の追加スカラー列。root.cols() =
	// scanCols ++ carriedCols なので、宣言count列はcarriedの後でなければ
	// expectedColumnsと列順が揃わない(§6.4のI-3参照)。P-04のcountCellsと
	// 同じ「使い捨てセル」パターンを踏襲する。
	declaredCountCells := make([]*sql.NullInt64, len(plan.declaredCounts))
	for i := range plan.declaredCounts {
		cell := new(sql.NullInt64)
		declaredCountCells[i] = cell
		targets = append(targets, cell)
	}

	bufs := make(map[int]bufAny, len(plan.sections))
	bySet := make(map[int]flatNode, len(plan.sections))
	secOff := make(map[int]int, len(plan.sections))
	secLen := make(map[int]int, len(plan.sections))
	// carriedBufs/carriedOff/carriedLen (P-05): the SAME shape as
	// bufs/secOff/secLen, one entry per section whose edge is multi-hop
	// (absent — not merely nil — for a single-hop section, so a lookup
	// miss and "no carried buffer" are the same thing below). A separate
	// map, not a second range tacked onto secOff/secLen, because the two
	// buffers are refreshed independently (issue P-26) and have
	// different Go types (Buf[C] vs Buf[P]).
	carriedBufs := make(map[int]bufAny, len(plan.sections))
	carriedOff := make(map[int]int, len(plan.sections))
	carriedLen := make(map[int]int, len(plan.sections))
	for _, n := range plan.sections {
		b := n.edge.child().newBufAny()
		bufs[n.set] = b
		bySet[n.set] = n
		off := len(targets)
		// n.edge.childScanCols() (NOT childCols(), and NOT
		// n.edge.child().cols()): the model buffer only ever knows how to
		// scan the CHILD table's own columns. A multi-hop edge's carried
		// column(s) are a DIFFERENT table's value (the parent's own join
		// key, read back via a JOIN — parts.go's Edge.Carried doc) and go
		// into the separate carried buffer below, never into b.
		targets = append(targets, b.targetsFor(n.edge.childScanCols())...)
		secOff[n.set] = off
		secLen[n.set] = len(targets) - off

		if cb := n.edge.newCarriedBufAny(); cb != nil {
			carriedBufs[n.set] = cb
			coff := len(targets)
			targets = append(targets, cb.targetsFor(n.edge.carriedScanCols())...)
			carriedOff[n.set] = coff
			carriedLen[n.set] = len(targets) - coff
		}
	}

	// Count sections occupy the tail of the layout: their join key columns
	// followed by their own _sl_cnt<set> column. The count column is
	// NULL-padded in every other branch, so it is scanned through a
	// sql.NullInt64 cell rather than a plain int64.
	countBufs := make(map[int]bufAny, len(plan.counts))
	countCells := make(map[int]*sql.NullInt64, len(plan.counts))
	countBySet := make(map[int]countPlan, len(plan.counts))
	for _, c := range plan.counts {
		var b bufAny
		var scanCols []Column
		if len(c.edge.sides()) > 1 {
			// R-24: keyCols は第1中間テーブルの実列名だが、その名前は
			// 最終子テーブルの Buf[C] には存在しない。PARENT の Buf[P] を
			// PARENT自身の列名(carriedScanCols)で読む。
			b = c.edge.newCarriedBufAny()
			scanCols = c.edge.carriedScanCols()
		} else {
			b = c.edge.child().newBufAny()
			scanCols = c.keyCols
		}
		countBufs[c.set] = b
		countBySet[c.set] = c
		targets = append(targets, b.targetsFor(scanCols)...)
		cell := new(sql.NullInt64)
		countCells[c.set] = cell
		targets = append(targets, cell)
	}

	// childrenOf[s] = sections whose parentSet is s, i.e. the edges that
	// will need a parent-key reading from a row belonging to set s.
	childrenOf := make(map[int][]flatNode, len(plan.sections)+1)
	for _, n := range plan.sections {
		childrenOf[n.parentSet] = append(childrenOf[n.parentSet], n)
	}

	// thensOf[s] = Then(...) boundaries whose parentSet is s — the same
	// idea as childrenOf, for the id-keyed thenKeys map instead.
	thensOf := make(map[int][]thenPlan, len(plan.thens))
	for _, tp := range plan.thens {
		thensOf[tp.parentSet] = append(thensOf[tp.parentSet], tp)
	}

	// countsOf[s] = count sections hanging under set s, i.e. the count
	// edges that need a parent-key reading from a row belonging to set s.
	countsOf := make(map[int][]countPlan, len(plan.counts))
	for _, c := range plan.counts {
		countsOf[c.parentSet] = append(countsOf[c.parentSet], c)
	}

	slots := resultSetSlots(plan)
	res.items = make([][]any, slots)
	res.parentKeys = make(map[int][]any, len(plan.sections)+1)
	res.childKeys = make(map[int][]any, len(plan.sections))
	res.thenKeys = make(map[int][]any, len(plan.thens))
	res.secOrds = make([][]sql.NullInt64, slots)
	res.counts = make(map[int]map[any]int64, len(plan.counts))
	for _, c := range plan.counts {
		res.counts[c.set] = make(map[any]int64)
	}

	// refreshBuf installs fresh's scan destinations over the [off, off+n)
	// slots of targets. It is called right after a row's model has been
	// built and its keys extracted, so the NEXT Scan writes into a brand
	// new buffer instead of the one the just-built model may still alias
	// (issue P-26).
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
	refreshBuf := func(fresh bufAny, cols []Column, off, n int) error {
		next := fresh.targetsFor(cols)
		if len(next) != n {
			return fmt.Errorf(
				"sectionload: refreshed scan buffer produced %d targets, want %d",
				len(next), n,
			)
		}
		copy(targets[off:off+n], next)
		return nil
	}

	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return islandResult{}, fmt.Errorf("sectionload: scanning row: %w", err)
		}
		if set == 0 {
			res.items[0] = append(res.items[0], rootBuf.buildFor(root.modelCols()))
			// P-13: 宣言済み count preload をこの行のモデルへ反映する。
			// §6.7の論証(name/typeのskewは1回のコード生成内で構造的に
			// 起こり得ない)が正しいなら、このアサーションは常に成立する
			// はず。だから失敗を黙ってスキップせず、明示エラーにする —
			// silent no-opにすると§6.7の前提が破れたときにそれを検出
			// できなくなり、論証自体が無意味になる。
			if len(plan.declaredCounts) > 0 {
				built := res.items[0][len(res.items[0])-1]
				loader, ok := built.(countPreloadable)
				if !ok {
					return islandResult{}, fmt.Errorf(
						"sectionload: %T does not implement countPreloadable, but a PreloadCount was declared on the parent query",
						built,
					)
				}
				for i, dc := range plan.declaredCounts {
					cell := declaredCountCells[i]
					if !cell.Valid {
						continue // 親行でNULLになることは通常無いが、安全側でスキップ(M-6: bobネイティブ実行のPreloadCountも相関サブクエリがNULLなら呼ばない契約と一致)
					}
					if err := loader.PreloadCount(dc.RelName, cell.Int64); err != nil {
						return islandResult{}, fmt.Errorf("sectionload: %s: %w", dc.RelName, err)
					}
				}
			}
			res.ords = append(res.ords, ord.Int64)
			for _, child := range childrenOf[0] {
				res.parentKeys[child.set] = append(res.parentKeys[child.set], cloneKey(child.edge.parentKeyFromBufAny(rootBuf)))
			}
			for _, tp := range thensOf[0] {
				res.thenKeys[tp.id] = append(res.thenKeys[tp.id], cloneKey(tp.edge.parentKeyFromBufAny(rootBuf)))
			}
			for _, cp := range countsOf[0] {
				res.parentKeys[cp.set] = append(res.parentKeys[cp.set], cloneKey(cp.edge.parentKeyFromBufAny(rootBuf)))
			}
			if rootChildKeyFn != nil {
				res.rootChildKeys = append(res.rootChildKeys, cloneKey(rootChildKeyFn(rowBufs{model: rootBuf, carried: rootCarriedBuf})))
			}
			// Every read of this row's root buffer is done by here (model
			// built, every key extracted and cloned), so retire it: the next
			// Scan must not be able to write through the model just appended
			// to res.items[0] (issue P-26).
			rootBuf = root.newBufAny()
			// root.scanCols(), not root.cols() — must match what was used
			// at setup time above, or the replacement buffer's target
			// count differs from rootLen (P-05 correction, same as
			// childScanCols() below for a section's own buffer).
			if err := refreshBuf(rootBuf, root.scanCols(), rootOff, rootLen); err != nil {
				return islandResult{}, err
			}
			// P-05: the root's carried buffer (only non-nil for a Then
			// boundary whose own edge is multi-hop) is retired the same
			// way, right after rootChildKeyFn above has read it.
			if rootCarriedBuf != nil {
				fresh := root.newCarriedBufAny()
				if err := refreshBuf(fresh, root.carriedScanCols(), rootCarriedOff, rootCarriedLen); err != nil {
					return islandResult{}, err
				}
				rootCarriedBuf = fresh
			}
			continue
		}
		if b, ok := bufs[int(set)]; ok {
			n := bySet[int(set)]
			res.items[int(set)] = append(res.items[int(set)], b.buildFor(n.edge.childModelCols()))
			res.childKeys[int(set)] = append(res.childKeys[int(set)], cloneKey(n.edge.childKeyFrom(rowBufs{model: b, carried: carriedBufs[int(set)]})))
			res.secOrds[int(set)] = append(res.secOrds[int(set)], ord)
			for _, child := range childrenOf[int(set)] {
				res.parentKeys[child.set] = append(res.parentKeys[child.set], cloneKey(child.edge.parentKeyFromBufAny(b)))
			}
			for _, tp := range thensOf[int(set)] {
				res.thenKeys[tp.id] = append(res.thenKeys[tp.id], cloneKey(tp.edge.parentKeyFromBufAny(b)))
			}
			for _, cp := range countsOf[int(set)] {
				res.parentKeys[cp.set] = append(res.parentKeys[cp.set], cloneKey(cp.edge.parentKeyFromBufAny(b)))
			}
			// Same as the root branch: this section's buffer has been fully
			// read, so retire it before the next Scan (issue P-26). Only
			// THIS section's buffer is replaced — every other section's
			// buffer merely took NULL padding on this row and still holds
			// nothing anyone kept.
			nb := n.edge.child().newBufAny()
			// n.edge.childScanCols() (NOT childCols(), and NOT
			// n.edge.child().cols()) must match what was used at setup time
			// or the replacement buffer's target count will differ — b only
			// ever scanned the child table's own columns (P-05); the carried
			// buffer (if any) is refreshed separately right below.
			if err := refreshBuf(nb, n.edge.childScanCols(), secOff[int(set)], secLen[int(set)]); err != nil {
				return islandResult{}, err
			}
			bufs[int(set)] = nb
			if _, ok := carriedBufs[int(set)]; ok {
				fresh := n.edge.newCarriedBufAny()
				if err := refreshBuf(fresh, n.edge.carriedScanCols(), carriedOff[int(set)], carriedLen[int(set)]); err != nil {
					return islandResult{}, err
				}
				carriedBufs[int(set)] = fresh
			}
			continue
		}
		if cp, ok := countBySet[int(set)]; ok {
			// A count row builds no model: only its grouped key and its
			// count(*) matter. A NULL key group (children whose nullable FK
			// is not set) matches no parent, so it is dropped here rather
			// than carried into the key map. mapKey normalizes (and, for
			// []byte, copies) kv immediately below, and the result is used
			// only as a map key within this same iteration — it never
			// survives to the next row's Scan, so cloneKey is unnecessary
			// here (unlike the key slices above, which are appended to
			// slices that persist across every remaining row). For the
			// same reason a count buffer is never retired the way a root
			// or section buffer is (issue P-26): a count row builds no
			// model, and nothing that outlives this iteration points into
			// that buffer.
			kv := cp.edge.childKeyFrom(countBufs[cp.set])
			if kv == nil {
				continue
			}
			k, err := mapKey(kv)
			if err != nil {
				return islandResult{}, fmt.Errorf("sectionload: %s count key: %w", cp.edge.relName(), err)
			}
			res.counts[cp.set][k] = countCells[cp.set].Int64
			continue
		}
		return islandResult{}, fmt.Errorf("sectionload: unexpected %s value %d", colSet, set)
	}
	if err := rows.Err(); err != nil {
		return islandResult{}, err
	}
	return res, nil
}

// resultSetSlots returns the number of slots res.items/res.secOrds need:
// one past the highest set number in use across row sections AND count
// sections. planIsland draws both from the same counter (interleaved in
// declaration order), so len(plan.sections)+1 undercounts whenever a count
// node is declared before a row section — the row section's set can then
// exceed len(plan.sections), and indexing res.items[set] would panic with
// index out of range (found during Task 3's review, 2026-08-13).
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
	// P-13: root.cols()(scanCols ++ carriedCols)の直後に宣言済み count
	// preload の alias を並べる。execIsland のscan target組み立て(root
	// 用のtargets追加)側もこの並びと揃える必要がある(§6.4のI-3参照)。
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
