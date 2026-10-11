package sectionload

// P-28: a Then(...) boundary nested under a non-root section has no set
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

type p28FakeRows struct {
	rowsData [][]any
	i        int
}

func (r *p28FakeRows) Columns() ([]string, error) { return nil, nil } // unused: execIsland only checks len()/name below, see fakeExecutor.cols
func (r *p28FakeRows) Next() bool                 { r.i++; return r.i <= len(r.rowsData) }
func (r *p28FakeRows) Err() error                 { return nil }
func (r *p28FakeRows) Close() error               { return nil }
func (r *p28FakeRows) Scan(dest ...any) error {
	row := r.rowsData[r.i-1]
	if len(dest) != len(row) {
		return fmt.Errorf("p28FakeRows.Scan: %d destinations, %d values", len(dest), len(row))
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
			return fmt.Errorf("p28FakeRows.Scan: unsupported dest type %T at position %d", d, i)
		}
	}
	return nil
}

// p28FakeCols carries the column names execIsland's positional layout
// check (expectedColumns) requires; wired in via a wrapper Rows so
// Columns() returns them without p28FakeRows needing to know its own
// position in the layout.
type p28ColsRows struct {
	*p28FakeRows
	cols []string
}

func (r p28ColsRows) Columns() ([]string, error) { return r.cols, nil }

type p28FakeExecutor struct{ rows scan.Rows }

func (e p28FakeExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return e.rows, nil
}

func (e p28FakeExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("p28FakeExecutor: ExecContext not implemented (not needed by execIsland)")
}

var _ bob.Executor = p28FakeExecutor{}

// --- fixtures: root -[Mid]-> section (set 1); Then(Leaf) under Mid ---

type (
	p28Root    struct{ ID int }
	p28RootBuf struct{ id int }
)

func (b *p28RootBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *p28RootBuf) BuildFor(modelCols []Column) p28Root {
	return p28Root{ID: b.id}
}

type (
	p28Mid    struct{ ID, RootID, OtherKey int }
	p28MidBuf struct{ id, rootID, otherKey int }
)

func (b *p28MidBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.rootID, &b.otherKey}
}

func (b *p28MidBuf) BuildFor(modelCols []Column) p28Mid {
	return p28Mid{ID: b.id, RootID: b.rootID, OtherKey: b.otherKey}
}

type (
	p28Leaf    struct{ ID, OtherKey int }
	p28LeafBuf struct{ id, otherKey int }
)

func (b *p28LeafBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.otherKey} }
func (b *p28LeafBuf) BuildFor(modelCols []Column) p28Leaf {
	return p28Leaf{ID: b.id, OtherKey: b.otherKey}
}

func p28RootMidEdge() Edge[p28Root, p28Mid] {
	return Edge[p28Root, p28Mid]{
		RelName:  "Mid",
		FromCols: []string{"id"},
		ToCols:   []string{"root_id"},
		ToMany:   false,
		Child: TableParts[p28Mid]{
			Table: "p28_mid",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "root_id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[p28Mid] { return &p28MidBuf{} },
		},
		ParentKey: func(p p28Root) []any { return []any{p.ID} },
		ChildKey:  func(c p28Mid) []any { return []any{c.RootID} },
		Stitch:    func(p p28Root, cs []p28Mid) error { return nil },
	}
}

func p28MidLeafEdge() Edge[p28Mid, p28Leaf] {
	return Edge[p28Mid, p28Leaf]{
		RelName:  "Leaf",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		ToMany:   true,
		Child: TableParts[p28Leaf]{
			Table: "p28_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[p28Leaf] { return &p28LeafBuf{} },
		},
		// The Then boundary's own key: deliberately NOT the same value as
		// EdgeA's parent key (root.id=100) — mid.other_key=999. If
		// execIsland ever hands back parentKeys[tp.parentSet] (the old,
		// buggy indexing) instead of thenKeys[tp.id], this test observes
		// 100, not 999.
		ParentKey: func(p p28Mid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c p28Leaf) []any { return []any{c.OtherKey} },
		Stitch:    func(p p28Mid, cs []p28Leaf) error { return nil },
	}
}

// TestExecIsland_ThenKeys_NotAliasedWithParentSectionSet is the direct
// regression test for issue P-28 (Codex review, 2026-08-09).
func TestExecIsland_ThenKeys_NotAliasedWithParentSectionSet(t *testing.T) {
	edgeA := p28RootMidEdge() // root -> mid, set 1
	edgeThen := p28MidLeafEdge()

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

	root := partsAdapter[p28Root]{TableParts[p28Root]{
		Table:   "p28_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
	}}

	// One root row (id=100), one section row (mid.id=200, mid.root_id=100,
	// mid.other_key=999). Row counts coincide (1 root row, 1 section row)
	// — exactly the condition under which the pre-fix bug stayed silent
	// instead of tripping runThen's length-mismatch guard.
	cols := expectedColumns(root, plan)
	rows := &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id, mid.id, mid.root_id, mid.other_key
			{int64(0), int64(1), 100, 0, 0, 0},
			{int64(1), nil, 0, 200, 100, 999},
		}},
	}
	// Fix up NULL padding types: root row's section-buffer columns and the
	// section row's ord are typed sql.NullInt64/int zero either way since
	// the fake Scan type-switches per destination, not per stored value —
	// the literal 0s above are fine as *int destinations regardless of row.

	exec := p28FakeExecutor{rows: rows}
	// The fake executor above ignores whatever SQL text bob.Build renders
	// and returns the canned rows regardless — any valid bob.Query will
	// do here, so a trivial RawQuery stands in for the real composed
	// island query this unit test does not need to construct.
	q := psql.RawQuery("SELECT 1")

	// P-05 signature-adapter only, no assertion change: execIsland's
	// rootChildKeyFn parameter changed from func(bufAny) []any to
	// func(rowBufs) []any (P-05 Task 5, so a multi-hop Then boundary's
	// root can also read its carried buffer). edgeA is single-hop, so
	// edgeA.parentKeyFromBufAny(r.model) is exactly what this test's
	// old direct call already exercised.
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
