package sectionload

// Regression tests for the interaction between ORDER BY row-reordering
// (ords/secOrds) and buffer-derived join keys (parentKeys/childKeys/
// thenKeys) inside the same functions. Reordering the model rows without
// reordering the keys aligned with them stitches a Then(...) boundary's
// results onto the wrong parent — silently, because nothing about the
// row counts changes.
//
// TestStitchIsland_ThenKeysFollowSectionOrdinalReorder covers a Then(...)
// nested directly under an ordered same-island section (stitchIsland's
// own resync). TestRunThen_ThenKeysFollowRootOrdinalReorder covers a
// Then(...) nested directly under another Then(...) boundary's own,
// ordered root (runThen's resync). Both are DB-free: a minimal fake
// bob.Executor returns canned rows; no SQL is actually parsed or run.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// --- shared minimal fixtures -------------------------------------------

type mrgRoot struct{ ID int }

type mrgMid struct {
	ID       int
	OtherKey int
	RootID   int // links to mrgRoot.ID for edgeMidThen (Test 2 only)
	Leaves   []*mrgLeaf
}
type mrgMidBuf struct{ id, otherKey, rootID int }

func (b *mrgMidBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.otherKey, &b.rootID}
}

func (b *mrgMidBuf) BuildFor(modelCols []Column) *mrgMid {
	return &mrgMid{ID: b.id, OtherKey: b.otherKey, RootID: b.rootID}
}

type (
	mrgLeaf    struct{ ID, OtherKey int }
	mrgLeafBuf struct{ id, otherKey int }
)

func (b *mrgLeafBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.otherKey}
}

func (b *mrgLeafBuf) BuildFor(modelCols []Column) *mrgLeaf {
	return &mrgLeaf{ID: b.id, OtherKey: b.otherKey}
}

// mrgMidLeafEdge is the Then(...) boundary shared by both tests: keyed by
// mid.OtherKey, deliberately distinct per mid row so a misaligned key
// picks up the wrong leaf.
func mrgMidLeafEdge() Edge[*mrgMid, *mrgLeaf] {
	return Edge[*mrgMid, *mrgLeaf]{
		RelName:  "Leaves",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		ToMany:   true,
		Child: TableParts[*mrgLeaf]{
			Table: "mrg_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[*mrgLeaf] { return &mrgLeafBuf{} },
		},
		ParentKey: func(p *mrgMid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c *mrgLeaf) []any { return []any{c.OtherKey} },
		Stitch: func(p *mrgMid, cs []*mrgLeaf) error {
			p.Leaves = cs
			return nil
		},
	}
}

// --- fake bob.Executor: canned rows, no DB, no real SQL execution ------

type mrgFakeRows struct {
	cols []string
	data [][]any
	pos  int
}

func newMrgFakeRows(cols []string, data [][]any) *mrgFakeRows {
	return &mrgFakeRows{cols: cols, data: data, pos: -1}
}

func (r *mrgFakeRows) Columns() ([]string, error) { return r.cols, nil }

func (r *mrgFakeRows) Next() bool {
	r.pos++
	return r.pos < len(r.data)
}

func (r *mrgFakeRows) Scan(dest ...any) error {
	row := r.data[r.pos]
	if len(dest) != len(row) {
		return fmt.Errorf("mrgFakeRows: dest count %d != row width %d", len(dest), len(row))
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *int64:
			*v = row[i].(int64)
		case *sql.NullInt64:
			if row[i] == nil {
				*v = sql.NullInt64{}
			} else {
				*v = row[i].(sql.NullInt64)
			}
		case *int:
			*v = row[i].(int)
		default:
			return fmt.Errorf("mrgFakeRows: unsupported dest type %T at position %d", d, i)
		}
	}
	return nil
}

func (r *mrgFakeRows) Close() error { return nil }
func (r *mrgFakeRows) Err() error   { return nil }

// mrgFakeExecutor returns one canned scan.Rows per call, in order; the
// actual SQL text is ignored entirely (bob.Build still renders it, but
// nothing here parses it).
type mrgFakeExecutor struct {
	calls   int
	perCall []*mrgFakeRows
}

func (e *mrgFakeExecutor) QueryContext(_ context.Context, _ string, _ ...any) (scan.Rows, error) {
	e.calls++
	if e.calls > len(e.perCall) {
		return nil, fmt.Errorf("mrgFakeExecutor: unexpected QueryContext call #%d", e.calls)
	}
	return e.perCall[e.calls-1], nil
}

func (e *mrgFakeExecutor) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	return nil, fmt.Errorf("mrgFakeExecutor: unexpected ExecContext call: %s", query)
}

var _ bob.Executor = &mrgFakeExecutor{}

// --- Test 1 (Critical, stitchIsland): same-island ordered section ------

// TestStitchIsland_ThenKeysFollowSectionOrdinalReorder is a direct,
// white-box test of stitchIsland: it must reorder thenKeys[tp.id] for any
// Then(...) boundary nested directly under a same-island section
// (tp.parentSet == n.set) using the exact same permutation it applies to
// items[n.set] via secOrds. Before the fix, thenKeys stayed in scan
// order while items[n.set] was reordered, so position i's key no longer
// belonged to position i's row.
//
// Uses 2 real (parent, child) pairs matched by ParentKey/ChildKey (not a
// no-op Stitch) so a parentKeys/childKeys misalignment introduced by a
// future change to the reorder logic would show up as the wrong child
// landing on the wrong parent, not just as a value the test never reads.
func TestStitchIsland_ThenKeysFollowSectionOrdinalReorder(t *testing.T) {
	stitched := map[int][]*mrgMid{}
	edgeMid := Edge[mrgRoot, *mrgMid]{
		RelName:  "Mids",
		FromCols: []string{"id"},
		ToCols:   []string{"id"}, // unused: parentKeys/childKeys are supplied explicitly below
		ToMany:   true,
		Child: TableParts[*mrgMid]{
			Table:   "mrg_mid",
			Columns: []Column{{Name: "id", DBType: "int4"}, {Name: "other_key", DBType: "int4"}},
			NewBuf:  func() Buf[*mrgMid] { return &mrgMidBuf{} },
		},
		ParentKey: func(p mrgRoot) []any { return []any{p.ID} },
		ChildKey:  func(c *mrgMid) []any { return []any{c.ID} },
		Stitch: func(p mrgRoot, cs []*mrgMid) error {
			stitched[p.ID] = cs
			return nil
		},
	}
	edgeLeaf := mrgMidLeafEdge()

	// midA links to root 10, midB links to root 20 — via edgeMid's own
	// ParentKey(root)=root.ID / ChildKey(mid)=mid.ID, so a parentKeys/
	// childKeys mixup (not just an items/thenKeys one) would visibly
	// attach the wrong mid to the wrong root below.
	midA := &mrgMid{ID: 10, OtherKey: 111}
	midB := &mrgMid{ID: 20, OtherKey: 222}

	sections := []flatNode{{set: 1, parentSet: 0, edge: edgeMid, hasKids: false}}
	thens := []thenPlan{{id: 99, parentSet: 1, edge: edgeLeaf, children: nil}}

	items := [][]any{
		{mrgRoot{ID: 10}, mrgRoot{ID: 20}}, // items[0]: 2 root rows
		{midA, midB},                       // items[1]: scan order [A, B]
	}
	parentKeys := map[int][]any{
		1: {[]any{10}, []any{20}}, // aligned with items[0]: each root's own key
	}
	childKeys := map[int][]any{
		1: {[]any{10}, []any{20}}, // aligned with items[1] scan order: mid.ID
	}
	// thenKeys[99] as execIsland would have built it: scan order, aligned
	// with items[1] BEFORE any reorder — [key(midA)=111, key(midB)=222].
	thenKeys := map[int][]any{
		99: {[]any{111}, []any{222}},
	}
	// secOrds[1]: midA's ordinal (2) sorts after midB's (1) — the desired
	// final order is [midB, midA], the reverse of scan order.
	secOrds := [][]sql.NullInt64{
		nil,
		{{Int64: 2, Valid: true}, {Int64: 1, Valid: true}},
	}

	plan := islandPlan{sections: sections, thens: thens}
	res := islandResult{items: items, parentKeys: parentKeys, childKeys: childKeys, thenKeys: thenKeys, secOrds: secOrds}
	if err := stitchIsland(plan, res); err != nil {
		t.Fatalf("stitchIsland: %v", err)
	}

	gotMids := items[1]
	if len(gotMids) != 2 || gotMids[0].(*mrgMid) != midB || gotMids[1].(*mrgMid) != midA {
		t.Fatalf("items[1] order: got %v, want [midB, midA] (by ascending ordinal)", gotMids)
	}

	if len(stitched[10]) != 1 || stitched[10][0] != midA {
		t.Fatalf("stitched[10] (root 10's mids): got %v, want [midA] — a parentKeys/childKeys "+
			"mixup around the reorder would attach the wrong mid here", stitched[10])
	}
	if len(stitched[20]) != 1 || stitched[20][0] != midB {
		t.Fatalf("stitched[20] (root 20's mids): got %v, want [midB]", stitched[20])
	}

	gotKeys := thenKeys[99]
	if len(gotKeys) != 2 {
		t.Fatalf("thenKeys[99]: got %d entries, want 2", len(gotKeys))
	}
	if kv, ok := gotKeys[0].([]any); !ok || len(kv) != 1 || kv[0] != 222 {
		t.Fatalf("thenKeys[99][0] = %#v, want []any{222} (midB's key, now in position 0) — "+
			"a stale scan-order key here means a Then(...) nested under this section would stitch "+
			"its results onto the wrong mid row", gotKeys[0])
	}
	if kv, ok := gotKeys[1].([]any); !ok || len(kv) != 1 || kv[0] != 111 {
		t.Fatalf("thenKeys[99][1] = %#v, want []any{111} (midA's key, now in position 1)", gotKeys[1])
	}
}

// TestStitchIsland_MissingThenKeysIsError guards the thenKeys sync invariant
// itself: stitchIsland must
// reject a thenKeys[tp.id] whose length disagrees with its section's row
// count even when no ORDER BY was requested (secOrds nil, idx == nil) — a
// mismatch here always means the sync invariant broke upstream, never "no
// reorder needed, so nothing to check."
func TestStitchIsland_MissingThenKeysIsError(t *testing.T) {
	edgeMid := Edge[mrgRoot, *mrgMid]{
		RelName:  "Mids",
		FromCols: []string{"id"},
		ToCols:   []string{"id"},
		ToMany:   true,
		Child: TableParts[*mrgMid]{
			Table:   "mrg_mid",
			Columns: []Column{{Name: "id", DBType: "int4"}, {Name: "other_key", DBType: "int4"}},
			NewBuf:  func() Buf[*mrgMid] { return &mrgMidBuf{} },
		},
		ParentKey: func(p mrgRoot) []any { return []any{p.ID} },
		ChildKey:  func(c *mrgMid) []any { return []any{c.ID} },
		Stitch:    func(p mrgRoot, cs []*mrgMid) error { return nil },
	}
	edgeLeaf := mrgMidLeafEdge()

	midA := &mrgMid{ID: 10, OtherKey: 111}
	midB := &mrgMid{ID: 20, OtherKey: 222}

	sections := []flatNode{{set: 1, parentSet: 0, edge: edgeMid, hasKids: false}}
	thens := []thenPlan{{id: 99, parentSet: 1, edge: edgeLeaf, children: nil}}

	items := [][]any{
		{mrgRoot{ID: 1}},
		{midA, midB},
	}
	parentKeys := map[int][]any{1: {[]any{1}}}
	childKeys := map[int][]any{1: {[]any{10}, []any{20}}}
	// thenKeys[99] has only 1 entry for 2 mids — a broken sync invariant,
	// not "no ORDER BY requested" (secOrds is nil below, so this is not
	// even about ordering).
	thenKeys := map[int][]any{99: {[]any{111}}}

	plan := islandPlan{sections: sections, thens: thens}
	res := islandResult{items: items, parentKeys: parentKeys, childKeys: childKeys, thenKeys: thenKeys}
	if err := stitchIsland(plan, res); err == nil {
		t.Fatal("stitchIsland: want error for thenKeys/section row count mismatch, got nil")
	}
}

// --- Test 2 (Critical, runThen): a Then(...) boundary's own root -------

// TestRunThen_ThenKeysFollowRootOrdinalReorder exercises runThen
// end-to-end (its real query-building and execIsland/stitchIsland path,
// with a fake executor standing in for the DB) for a Then(...) boundary
// whose own edge requests ORDER BY and which has a second Then(...)
// nested directly under its root. Before the fix, the nested boundary's
// thenKeysBySet stayed in scan order while runThen reordered
// items[0]/childKeys — so the wrong leaf attached to the wrong mid.
//
// Uses 2 real roots matched to their mids by RootID (not a constant
// ParentKey/ChildKey) so a rootChildKeys/childKeys misalignment introduced
// by a future change to runThen's reorder logic would show up as the
// wrong mid landing on the wrong root, not just as a value the test never
// reads.
func TestRunThen_ThenKeysFollowRootOrdinalReorder(t *testing.T) {
	edgeLeaf := mrgMidLeafEdge()

	stitchedByRoot := map[int][]*mrgMid{}
	edgeMidThen := Edge[mrgRoot, *mrgMid]{
		RelName:  "MidsThen",
		FromCols: []string{"id"},
		ToCols:   []string{"root_id"},
		ToMany:   true,
		Options:  Mods(sm.OrderBy("other_key")),
		Child: TableParts[*mrgMid]{
			Table: "mrg_mid",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
				{Name: "root_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*mrgMid] { return &mrgMidBuf{} },
		},
		ParentKey: func(p mrgRoot) []any { return []any{p.ID} },
		ChildKey:  func(c *mrgMid) []any { return []any{c.RootID} },
		Stitch: func(p mrgRoot, cs []*mrgMid) error {
			stitchedByRoot[p.ID] = cs
			return nil
		},
	}

	tp := thenPlan{
		id:        1,
		parentSet: 0,
		edge:      edgeMidThen,
		children:  []Node{Then(NewNode(edgeLeaf))},
	}

	// 2 roots: root 1 owns midA (RootID=1), root 2 owns midB (RootID=2).
	parents := []any{mrgRoot{ID: 1}, mrgRoot{ID: 2}}

	// Call #1: edgeMidThen's own island root query (route 4, via
	// buildEdgeSelect/composeIsland) — 2 mid rows, scan order [A, B],
	// _sl_ord = the ordinal buildEdgeSelect's collapseOrderBy captured:
	// A=2, B=1, so ascending order is [B, A].
	rowsThen := newMrgFakeRows(
		[]string{"_sl_set", "_sl_ord", "id", "other_key", "root_id"},
		[][]any{
			{int64(0), sql.NullInt64{Int64: 2, Valid: true}, 10, 111, 1}, // midA, root 1
			{int64(0), sql.NullInt64{Int64: 1, Valid: true}, 20, 222, 2}, // midB, root 2
		},
	)
	// Call #2: the nested Then(Leaves) boundary's own query — returns
	// both leaves regardless of which keys were actually requested (this
	// fake ignores SQL text/args), so the test isolates the STITCH-back
	// step, which is where a stale key would attach the wrong leaf.
	rowsLeaf := newMrgFakeRows(
		[]string{"_sl_set", "_sl_ord", "id", "other_key"},
		[][]any{
			{int64(0), sql.NullInt64{}, 1, 111}, // leaf for midA
			{int64(0), sql.NullInt64{}, 2, 222}, // leaf for midB
		},
	)
	exec := &mrgFakeExecutor{perCall: []*mrgFakeRows{rowsThen, rowsLeaf}}

	if err := runThen(context.Background(), exec, tp, parents, nil); err != nil {
		t.Fatalf("runThen: %v", err)
	}

	// Outer check: root 1 got midA (RootID=1), root 2 got midB (RootID=2),
	// regardless of the ORDER BY reorder — a rootChildKeys/childKeys
	// mixup around that reorder would attach the wrong mid to the wrong
	// root here.
	if len(stitchedByRoot[1]) != 1 || stitchedByRoot[1][0].ID != 10 {
		t.Fatalf("stitchedByRoot[1] (root 1's mids): got %v, want [midA(ID=10)]", stitchedByRoot[1])
	}
	if len(stitchedByRoot[2]) != 1 || stitchedByRoot[2][0].ID != 20 {
		t.Fatalf("stitchedByRoot[2] (root 2's mids): got %v, want [midB(ID=20)]", stitchedByRoot[2])
	}

	// Inner check: each mid got its own leaf (key = mid.OtherKey), not the
	// other mid's — the thenKeysBySet resync this test exists to cover.
	for rootID, mids := range stitchedByRoot {
		for _, mid := range mids {
			if len(mid.Leaves) != 1 {
				t.Fatalf("root %d, mid %d: got %d leaves, want 1: %v", rootID, mid.ID, len(mid.Leaves), mid.Leaves)
			}
			if mid.Leaves[0].OtherKey != mid.OtherKey {
				t.Fatalf("root %d, mid %d (other_key=%d) got leaf other_key=%d — a stale thenKeysBySet "+
					"entry attached the wrong leaf (this is exactly what the pre-fix bug produces once "+
					"the root is reordered by ORDER BY)", rootID, mid.ID, mid.OtherKey, mid.Leaves[0].OtherKey)
			}
		}
	}
}
