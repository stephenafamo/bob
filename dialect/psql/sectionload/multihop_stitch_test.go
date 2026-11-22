package sectionload

// A fake-executor (no DB) test that drives execIsland and stitchIsland end
// to end for a 2-hop edge, to verify that the carried-key dispatch actually
// stitches correctly:
//
//   - the carried key value (read from the SEPARATE carried buffer, via
//     childKeyFrom) is what groups children under their parent — not
//     anything read from the final table's own model columns;
//   - the final child table (multiHopLeaf) has no column holding the parent's
//     key at all, and stitching still succeeds (the whole premise of multi-hop edges);
//   - a NULL-padded row belonging to a DIFFERENT set (here: the root row,
//     set 0) never contributes a bogus entry to the multi-hop section's
//     own childKeys — verified by asserting childKeys[1] has EXACTLY as
//     many entries as leaf rows, each carrying the correct value.
//
// A dedicated fake Rows/Executor is defined here rather than reusing/
// extending thenKeyFakeRows, whose existing types are left untouched — this
// one additionally supports *string destinations, which thenKeyFakeRows'
// Scan does not.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/scan"
)

type multiHopFakeRows struct {
	cols     []string
	rowsData [][]any
	i        int
}

func (r *multiHopFakeRows) Columns() ([]string, error) { return r.cols, nil }
func (r *multiHopFakeRows) Next() bool                 { r.i++; return r.i <= len(r.rowsData) }
func (r *multiHopFakeRows) Err() error                 { return nil }
func (r *multiHopFakeRows) Close() error               { return nil }

func (r *multiHopFakeRows) Scan(dest ...any) error {
	row := r.rowsData[r.i-1]
	if len(dest) != len(row) {
		return fmt.Errorf("multiHopFakeRows.Scan: %d destinations, %d values", len(dest), len(row))
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *int64:
			if row[i] == nil {
				*v = 0
			} else {
				*v = row[i].(int64)
			}
		case *sql.NullInt64:
			if row[i] == nil {
				*v = sql.NullInt64{}
			} else {
				*v = sql.NullInt64{Int64: row[i].(int64), Valid: true}
			}
		case *int:
			if row[i] == nil {
				*v = 0
			} else {
				*v = row[i].(int)
			}
		case *string:
			if row[i] == nil {
				*v = ""
			} else {
				*v = row[i].(string)
			}
		default:
			return fmt.Errorf("multiHopFakeRows.Scan: unsupported dest type %T at position %d", d, i)
		}
	}
	return nil
}

type multiHopFakeExecutor struct{ rows scan.Rows }

func (e multiHopFakeExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return e.rows, nil
}

func (e multiHopFakeExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("multiHopFakeExecutor: ExecContext not implemented (not needed by execIsland)")
}

var _ bob.Executor = multiHopFakeExecutor{}

func TestMultiHopStitch_CarriedKeyGroupsChildrenUnderParent(t *testing.T) {
	var stitchedParent multiHopParent
	var stitchedChildren []multiHopLeaf
	edge := multiHopTwoHopEdge()
	edge.Stitch = func(p multiHopParent, cs []multiHopLeaf) error {
		stitchedParent = p
		stitchedChildren = cs
		return nil
	}

	tree := []Node{NewNode(edge)}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 || plan.sections[0].set != 1 {
		t.Fatalf("expected exactly one section with set=1, got %+v", plan.sections)
	}

	root := partsAdapter[multiHopParent]{multiHopParentParts()}
	cols := expectedColumns(root, plan)

	// Column layout: _sl_set, _sl_ord, root.id, leaf.id, leaf.label,
	// _sl_k0 (carried). Row 0 is the root (parent id=1); rows 1-2 are two
	// leaves, both carrying _sl_k0=1 — the SAME value the root's own
	// ParentKey (root.id) produces, so they must stitch under this root.
	// Row 0's own leaf-section columns (id/label/_sl_k0) are NULL-padded,
	// exactly like a real UNION ALL branch would pad them; this pins that
	// the root row never contributes a bogus childKeys[1] entry from that
	// padding.
	rows := &multiHopFakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, nil, nil, nil},
			{int64(1), nil, nil, 10, "a", 1},
			{int64(1), nil, nil, 11, "b", 1},
		},
	}
	exec := multiHopFakeExecutor{rows: rows}
	q := psql.RawQuery("SELECT 1") // ignored by the fake executor

	res, err := execIsland(context.Background(), exec, q, root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}

	// The final table (multiHopLeaf) has no field for the parent's key at all
	// — BuildFor never sees it — yet both leaves are still correctly
	// scanned.
	if len(res.items[1]) != 2 {
		t.Fatalf("items[1]: got %d leaves, want 2: %+v", len(res.items[1]), res.items[1])
	}
	leaf0, ok := res.items[1][0].(multiHopLeaf)
	if !ok || leaf0.ID != 10 || leaf0.Label != "a" {
		t.Fatalf("items[1][0] = %#v, want multiHopLeaf{ID:10, Label:\"a\"}", res.items[1][0])
	}
	leaf1, ok := res.items[1][1].(multiHopLeaf)
	if !ok || leaf1.ID != 11 || leaf1.Label != "b" {
		t.Fatalf("items[1][1] = %#v, want multiHopLeaf{ID:11, Label:\"b\"}", res.items[1][1])
	}

	// childKeys[1] must have EXACTLY 2 entries (one per leaf row) — not 3
	// — confirming the root row's NULL-padded leaf-section columns
	// (including _sl_k0) contributed nothing here. Each entry's value
	// must be 1: read from the SEPARATE carried buffer via childKeyFrom,
	// never from leaf.ID/leaf.Label (which would be 10/11, not 1).
	if len(res.childKeys[1]) != 2 {
		t.Fatalf("childKeys[1]: got %d entries, want 2: %v", len(res.childKeys[1]), res.childKeys[1])
	}
	for i, want := range []int{1, 1} {
		kv, ok := res.childKeys[1][i].([]any)
		if !ok || len(kv) != 1 || kv[0] != want {
			t.Fatalf("childKeys[1][%d] = %#v, want []any{%d} (the carried key, not the leaf's own id)", i, res.childKeys[1][i], want)
		}
	}

	// parentKeys[1] (the root's own key, as this edge sees it) must align
	// with items[0] (one root row, id=1).
	if len(res.parentKeys[1]) != 1 {
		t.Fatalf("parentKeys[1]: got %d entries, want 1: %v", len(res.parentKeys[1]), res.parentKeys[1])
	}
	pk, ok := res.parentKeys[1][0].([]any)
	if !ok || len(pk) != 1 || pk[0] != 1 {
		t.Fatalf("parentKeys[1][0] = %#v, want []any{1}", res.parentKeys[1][0])
	}

	// Stitch bottom-up and confirm the parent actually receives both
	// children — the end-to-end proof that the carried key (not a
	// final-table column) is what drove the grouping.
	if err := stitchIsland(plan, res); err != nil {
		t.Fatalf("stitchIsland: %v", err)
	}
	if stitchedParent.ID != 1 {
		t.Fatalf("Stitch called with parent.ID = %d, want 1", stitchedParent.ID)
	}
	if len(stitchedChildren) != 2 {
		t.Fatalf("Stitch called with %d children, want 2: %+v", len(stitchedChildren), stitchedChildren)
	}
}
