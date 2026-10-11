package sectionload

// A Then(...) boundary nested under a non-root section has no set
// number of its own — only a parentSet, the set number of the section it
// hangs under. Before this fix, execIsland was queried for such a
// boundary's parent keys via parentKeys[tp.parentSet], but parentKeys is
// keyed by CHILD SECTION set numbers, not by "whoever's parentSet this
// is". For a Then boundary nested directly under section S (S's own set
// number is exactly tp.parentSet), this reads parentKeys[S.set] — which
// holds the key S's OWN edge extracts from S's PARENT rows, not the key
// the Then boundary's edge extracts from S's OWN rows. When the row
// counts happen to coincide (as they do here — one root row, one section
// row), the length-mismatch guard never fires, and the wrong values are
// used silently.
//
// This test builds exactly that shape (root -[Mid]-> section, set 1;
// Then(Leaf) nested under Mid, parentSet 1) with deliberately different
// key values on the two edges (root.id=100 vs mid.other_key=999), and a
// fake bob.Executor (canned rows, no DB) so execIsland's thenKeys can be
// asserted directly: it must equal mid.other_key (999), never root.id
// (100), which is what the pre-fix parentKeys[tp.parentSet] would have
// held here.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/scan"
)

// --- fake bob.Executor: canned rows, no DB, no real SQL execution ---
// execIsland only needs whatever bob.Build(ctx, q) renders to pass
// through to exec.QueryContext(ctx, sqlStr, args...); this fake ignores
// both and returns pre-built rows matching the fixed column layout
// expectedColumns(root, plan) produces.

type thenKeyFakeRows struct {
	rowsData [][]any
	i        int
}

func (r *thenKeyFakeRows) Columns() ([]string, error) { return nil, nil } // unused: execIsland only checks len()/name below, see fakeExecutor.cols
func (r *thenKeyFakeRows) Next() bool                 { r.i++; return r.i <= len(r.rowsData) }
func (r *thenKeyFakeRows) Err() error                 { return nil }
func (r *thenKeyFakeRows) Close() error               { return nil }
func (r *thenKeyFakeRows) Scan(dest ...any) error {
	row := r.rowsData[r.i-1]
	if len(dest) != len(row) {
		return fmt.Errorf("thenKeyFakeRows.Scan: %d destinations, %d values", len(dest), len(row))
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *int64:
			*v = row[i].(int64)
		case *sql.NullInt64:
			if row[i] == nil {
				*v = sql.NullInt64{}
			} else {
				*v = sql.NullInt64{Int64: row[i].(int64), Valid: true}
			}
		case *int:
			*v = row[i].(int)
		default:
			return fmt.Errorf("thenKeyFakeRows.Scan: unsupported dest type %T at position %d", d, i)
		}
	}
	return nil
}

// thenKeyFakeCols carries the column names execIsland's positional layout
// check (expectedColumns) requires; wired in via a wrapper Rows so
// Columns() returns them without thenKeyFakeRows needing to know its own
// position in the layout.
type thenKeyColsRows struct {
	*thenKeyFakeRows
	cols []string
}

func (r thenKeyColsRows) Columns() ([]string, error) { return r.cols, nil }

type thenKeyFakeExecutor struct{ rows scan.Rows }

func (e thenKeyFakeExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return e.rows, nil
}

func (e thenKeyFakeExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("thenKeyFakeExecutor: ExecContext not implemented (not needed by execIsland)")
}

var _ bob.Executor = thenKeyFakeExecutor{}

// --- fixtures: root -[Mid]-> section (set 1); Then(Leaf) under Mid ---

type (
	thenKeyRoot    struct{ ID int }
	thenKeyRootBuf struct{ id int }
)

func (b *thenKeyRootBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *thenKeyRootBuf) BuildFor(modelCols []Column) thenKeyRoot {
	return thenKeyRoot{ID: b.id}
}

type (
	thenKeyMid    struct{ ID, RootID, OtherKey int }
	thenKeyMidBuf struct{ id, rootID, otherKey int }
)

func (b *thenKeyMidBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.rootID, &b.otherKey}
}

func (b *thenKeyMidBuf) BuildFor(modelCols []Column) thenKeyMid {
	return thenKeyMid{ID: b.id, RootID: b.rootID, OtherKey: b.otherKey}
}

type (
	thenKeyLeaf    struct{ ID, OtherKey int }
	thenKeyLeafBuf struct{ id, otherKey int }
)

func (b *thenKeyLeafBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.otherKey} }
func (b *thenKeyLeafBuf) BuildFor(modelCols []Column) thenKeyLeaf {
	return thenKeyLeaf{ID: b.id, OtherKey: b.otherKey}
}

func thenKeyRootMidEdge() Edge[thenKeyRoot, thenKeyMid] {
	return Edge[thenKeyRoot, thenKeyMid]{
		RelName:  "Mid",
		FromCols: []string{"id"},
		ToCols:   []string{"root_id"},
		ToMany:   false,
		Child: TableParts[thenKeyMid]{
			Table: "tk_mid",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "root_id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[thenKeyMid] { return &thenKeyMidBuf{} },
		},
		ParentKey: func(p thenKeyRoot) []any { return []any{p.ID} },
		ChildKey:  func(c thenKeyMid) []any { return []any{c.RootID} },
		Stitch:    func(p thenKeyRoot, cs []thenKeyMid) error { return nil },
	}
}

func thenKeyMidLeafEdge() Edge[thenKeyMid, thenKeyLeaf] {
	return Edge[thenKeyMid, thenKeyLeaf]{
		RelName:  "Leaf",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		ToMany:   true,
		Child: TableParts[thenKeyLeaf]{
			Table: "tk_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[thenKeyLeaf] { return &thenKeyLeafBuf{} },
		},
		// The Then boundary's own key: deliberately NOT the same value as
		// EdgeA's parent key (root.id=100) — mid.other_key=999. If
		// execIsland ever hands back parentKeys[tp.parentSet] (the old,
		// buggy indexing) instead of thenKeys[tp.id], this test observes
		// 100, not 999.
		ParentKey: func(p thenKeyMid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c thenKeyLeaf) []any { return []any{c.OtherKey} },
		Stitch:    func(p thenKeyMid, cs []thenKeyLeaf) error { return nil },
	}
}

// TestExecIsland_ThenKeys_NotAliasedWithParentSectionSet is the direct
// regression test for the parentSet / set-number aliasing described at the
// top of this file.
func TestExecIsland_ThenKeys_NotAliasedWithParentSectionSet(t *testing.T) {
	edgeA := thenKeyRootMidEdge() // root -> mid, set 1
	edgeThen := thenKeyMidLeafEdge()

	tree := []Node{NewNode(edgeA, Then(NewNode(edgeThen)))}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	if len(plan.sections) != 1 || plan.sections[0].set != 1 {
		t.Fatalf("expected exactly one section with set=1, got %+v", plan.sections)
	}
	if len(plan.thens) != 1 || plan.thens[0].parentSet != 1 {
		t.Fatalf("expected exactly one Then boundary with parentSet=1, got %+v", plan.thens)
	}
	thenID := plan.thens[0].id
	if thenID == 1 {
		t.Fatalf("thenPlan.id must not collide with the section's own set (1); got %d", thenID)
	}

	root := partsAdapter[thenKeyRoot]{TableParts[thenKeyRoot]{
		Table:   "tk_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[thenKeyRoot] { return &thenKeyRootBuf{} },
	}}

	// One root row (id=100), one section row (mid.id=200, mid.root_id=100,
	// mid.other_key=999). Row counts coincide (1 root row, 1 section row)
	// — exactly the condition under which the pre-fix bug stayed silent
	// instead of tripping runThen's length-mismatch guard.
	cols := expectedColumns(root, plan)
	rows := &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id, mid.id, mid.root_id, mid.other_key
			{int64(0), int64(1), 100, 0, 0, 0},
			{int64(1), nil, 0, 200, 100, 999},
		}},
	}
	// Fix up NULL padding types: root row's section-buffer columns and the
	// section row's ord are typed sql.NullInt64/int zero either way since
	// the fake Scan type-switches per destination, not per stored value —
	// the literal 0s above are fine as *int destinations regardless of row.

	exec := thenKeyFakeExecutor{rows: rows}
	// The fake executor above ignores whatever SQL text bob.Build renders
	// and returns the canned rows regardless — any valid bob.Query will
	// do here, so a trivial RawQuery stands in for the real composed
	// island query this unit test does not need to construct.
	q := psql.RawQuery("SELECT 1")

	// execIsland's rootChildKeyFn takes func(rowBufs) []any so that a
	// multi-hop Then boundary's root can also read its carried buffer.
	// edgeA is single-hop, so edgeA.parentKeyFromBufAny(r.model) is all
	// that is needed here.
	res, err := execIsland(context.Background(), exec, q, root, plan,
		func(r rowBufs) []any { return edgeA.parentKeyFromBufAny(r.model) })
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	thenKeys, rootChildKeys := res.thenKeys, res.rootChildKeys

	got := thenKeys[thenID]
	if len(got) != 1 {
		t.Fatalf("thenKeys[%d]: got %d entries, want 1: %v", thenID, len(got), got)
	}
	gotKV, ok := got[0].([]any)
	if !ok || len(gotKV) != 1 {
		t.Fatalf("thenKeys[%d][0]: got %#v, want a 1-element []any", thenID, got[0])
	}
	if gotKV[0] != 999 {
		t.Fatalf("thenKeys[%d][0] = %v, want 999 (mid.other_key) — got root.id (100) would mean the pre-fix aliasing bug is back", thenID, gotKV[0])
	}

	// rootChildKeys: the parent-supplied rootChildKeyFn (edgeA's own
	// parentKeyFromBufAny, standing in for whatever runThen would pass)
	// must be evaluated against ROOT rows, one entry per root row.
	if len(rootChildKeys) != 1 {
		t.Fatalf("rootChildKeys: got %d entries, want 1: %v", len(rootChildKeys), rootChildKeys)
	}
	rootKV, ok := rootChildKeys[0].([]any)
	if !ok || len(rootKV) != 1 || rootKV[0] != 100 {
		t.Fatalf("rootChildKeys[0] = %#v, want []any{100}", rootChildKeys[0])
	}
}
