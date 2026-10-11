package sectionload

// execIsland must not share one scan buffer across rows. Generated BuildFor
// copies buffer fields by value, so slice-typed fields such as
// json.RawMessage only copy the slice header, and the final
// json.RawMessage.UnmarshalJSON (*m = append((*m)[:0], data...)) writes into
// the existing backing array in place. A shared buffer would let the next
// row's Scan overwrite the previous row's model.
//
// These tests reproduce that without a DB: bufAliasFakeRows.Scan does the same
// in-place append. The two payloads must have the same byte length, or
// append reallocates and a buggy implementation would pass; the helper
// bufAliasRequireSameLength checks that precondition.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// --- fixtures: root and child with a jsonb-like column ---

type bufAliasRoot struct {
	ID   int
	Blob json.RawMessage
}

type bufAliasRootBuf struct {
	id   int
	blob json.RawMessage
}

func (b *bufAliasRootBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.blob} }

// BuildFor is a plain value copy, like generated code. Blob is a slice, so
// only the header is copied and the backing array stays shared with the
// buffer.
func (b *bufAliasRootBuf) BuildFor(modelCols []Column) bufAliasRoot {
	return bufAliasRoot{ID: b.id, Blob: b.blob}
}

type bufAliasChild struct {
	ID       int
	ParentID int
	Blob     json.RawMessage
}

type bufAliasChildBuf struct {
	id, parentID int
	blob         json.RawMessage
}

func (b *bufAliasChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.parentID, &b.blob}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "parent_id":
			out[i] = &b.parentID
		case "blob":
			out[i] = &b.blob
		default:
			panic("bufAliasChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *bufAliasChildBuf) BuildFor(modelCols []Column) bufAliasChild {
	return bufAliasChild{ID: b.id, ParentID: b.parentID, Blob: b.blob}
}

func bufAliasRootParts() TableParts[bufAliasRoot] {
	return TableParts[bufAliasRoot]{
		Table: "ba_root",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "blob", DBType: "jsonb"},
		},
		NewBuf: func() Buf[bufAliasRoot] { return &bufAliasRootBuf{} },
	}
}

func bufAliasEdge() Edge[bufAliasRoot, bufAliasChild] {
	return Edge[bufAliasRoot, bufAliasChild]{
		RelName:  "Children",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Child: TableParts[bufAliasChild]{
			Table: "ba_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "blob", DBType: "jsonb"},
			},
			NewBuf: func() Buf[bufAliasChild] { return &bufAliasChildBuf{} },
		},
		ParentKey: func(p bufAliasRoot) []any { return []any{p.ID} },
		ChildKey:  func(c bufAliasChild) []any { return []any{c.ParentID} },
		Stitch:    func(p bufAliasRoot, cs []bufAliasChild) error { return nil },
	}
}

// --- fake rows: Scan that writes into *json.RawMessage in place ---
//
// Same role as thenKeyFakeRows (thenkey_test.go), which only handles
// int/int64 destinations. This one needs a destination that appends to the
// existing backing array, so it is dedicated.

type bufAliasFakeRows struct {
	cols     []string
	rowsData [][]any
	i        int
}

func (r *bufAliasFakeRows) Columns() ([]string, error) { return r.cols, nil }
func (r *bufAliasFakeRows) Next() bool                 { r.i++; return r.i <= len(r.rowsData) }
func (r *bufAliasFakeRows) Err() error                 { return nil }
func (r *bufAliasFakeRows) Close() error               { return nil }

func (r *bufAliasFakeRows) Scan(dest ...any) error {
	row := r.rowsData[r.i-1]
	if len(dest) != len(row) {
		return fmt.Errorf("bufAliasFakeRows.Scan: %d destinations, %d values", len(dest), len(row))
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
		case *json.RawMessage:
			// Mimics the standard library: json.RawMessage.UnmarshalJSON
			// writes into the existing backing array via
			// *m = append((*m)[:0], data...). A nil row[i] (NULL padding for
			// other sections) only truncates to length 0, as in the real one.
			b, _ := row[i].([]byte)
			*v = append((*v)[:0], b...)
		default:
			return fmt.Errorf("bufAliasFakeRows.Scan: unsupported dest type %T at position %d", d, i)
		}
	}
	return nil
}

// bufAliasPayloadA/B are distinct JSON values of the same byte length. The equal
// length is required: if the second were longer, append would reallocate
// and a buggy implementation would still pass.
var (
	bufAliasPayloadA = []byte(`{"row":"a"}`)
	bufAliasPayloadB = []byte(`{"row":"b"}`)
)

func bufAliasRequireSameLength(t *testing.T) {
	t.Helper()
	if len(bufAliasPayloadA) != len(bufAliasPayloadB) {
		t.Fatalf(
			"bufAliasPayloadA/B differ in byte length (%d vs %d); append would allocate a new array, "+
				"so a buggy implementation that reuses the buffer would pass and the check would be meaningless",
			len(bufAliasPayloadA), len(bufAliasPayloadB),
		)
	}
}

// --- tests ---

// TestExecIsland_SectionBufferNotAliasedAcrossRows: when one section returns
// two rows, the first row's child model JSON must not turn into the second
// row's value.
func TestExecIsland_SectionBufferNotAliasedAcrossRows(t *testing.T) {
	bufAliasRequireSameLength(t)

	plan, err := planIsland([]Node{NewNode(bufAliasEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 {
		t.Fatalf("sections: got %d, want 1", len(plan.sections))
	}
	set := plan.sections[0].set

	root := partsAdapter[bufAliasRoot]{bufAliasRootParts()}
	cols := expectedColumns(root, plan)

	// layout: _sl_set, _sl_ord, root.id, root.blob,
	//         child.id, child.parent_id, child.blob
	exec := thenKeyFakeExecutor{rows: &bufAliasFakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, []byte(`{}`), 0, 0, nil},
			{int64(1), nil, 0, nil, 10, 1, bufAliasPayloadA},
			{int64(1), nil, 0, nil, 11, 1, bufAliasPayloadB},
		},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[set]) != 2 {
		t.Fatalf("section items: got %d, want 2", len(res.items[set]))
	}

	first := res.items[set][0].(bufAliasChild)
	second := res.items[set][1].(bufAliasChild)
	if string(first.Blob) != string(bufAliasPayloadA) {
		t.Fatalf(
			"first child model JSON = %q, want %q: the section buffer is not recreated per row, "+
				"so the second row's Scan overwrote the first model's backing array",
			first.Blob, bufAliasPayloadA,
		)
	}
	if string(second.Blob) != string(bufAliasPayloadB) {
		t.Fatalf("second child model JSON = %q, want %q", second.Blob, bufAliasPayloadB)
	}
}

// TestExecIsland_RootBufferNotAliasedAcrossRows: the same for two rows from
// the root (set 0), i.e. a table with a jsonb column acting as the parent.
func TestExecIsland_RootBufferNotAliasedAcrossRows(t *testing.T) {
	bufAliasRequireSameLength(t)

	plan, err := planIsland([]Node{NewNode(bufAliasEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := partsAdapter[bufAliasRoot]{bufAliasRootParts()}
	cols := expectedColumns(root, plan)

	exec := thenKeyFakeExecutor{rows: &bufAliasFakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, bufAliasPayloadA, 0, 0, nil},
			{int64(0), int64(2), 2, bufAliasPayloadB, 0, 0, nil},
		},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[0]) != 2 {
		t.Fatalf("root items: got %d, want 2", len(res.items[0]))
	}

	first := res.items[0][0].(bufAliasRoot)
	second := res.items[0][1].(bufAliasRoot)
	if string(first.Blob) != string(bufAliasPayloadA) {
		t.Fatalf(
			"first parent model JSON = %q, want %q: the root buffer is not recreated per row, "+
				"so the second row's Scan overwrote the first model's backing array",
			first.Blob, bufAliasPayloadA,
		)
	}
	if string(second.Blob) != string(bufAliasPayloadB) {
		t.Fatalf("second parent model JSON = %q, want %q", second.Blob, bufAliasPayloadB)
	}
}

// bufAliasEdgeColumnPlanRestricted is bufAliasEdge with a ColumnPlan that keeps
// id/blob in Model and pushes parent_id (the join key) to Internal —
// childCols() then returns id, blob, parent_id, a DIFFERENT order than
// the table's declared id, parent_id, blob.
func bufAliasEdgeColumnPlanRestricted(t *testing.T) Edge[bufAliasRoot, bufAliasChild] {
	t.Helper()
	e := bufAliasEdge()
	plan, err := NewColumnPlan(
		e.Child.Columns,
		[]string{"id", "blob"},
		nil,
		[]string{"parent_id"},
	)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	e.ColumnPlan = plan
	return e
}

// TestExecIsland_SectionBufferNotAliasedAcrossRows_ColumnPlanRestricted is
// the ColumnPlan-restricted counterpart of
// TestExecIsland_SectionBufferNotAliasedAcrossRows: refreshBuf's caller uses n.edge.childCols(), which
// under a ColumnPlan is Model++Internal — a different column order than
// n.edge.child().cols() (the table's declared order). This test exercises
// that reordered path instead of only the no-ColumnPlan case above.
func TestExecIsland_SectionBufferNotAliasedAcrossRows_ColumnPlanRestricted(t *testing.T) {
	bufAliasRequireSameLength(t)

	edge := bufAliasEdgeColumnPlanRestricted(t)
	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 {
		t.Fatalf("sections: got %d, want 1", len(plan.sections))
	}
	set := plan.sections[0].set

	root := partsAdapter[bufAliasRoot]{bufAliasRootParts()}
	cols := expectedColumns(root, plan)

	// layout: _sl_set, _sl_ord, root.id, root.blob,
	//         child.id, child.blob, child.parent_id
	// (childCols() order under this ColumnPlan is Model[id,blob] then
	// Internal[parent_id] — NOT the table's declared id,parent_id,blob.)
	exec := thenKeyFakeExecutor{rows: &bufAliasFakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, []byte(`{}`), 0, nil, 0},
			{int64(1), nil, 0, nil, 10, bufAliasPayloadA, 1},
			{int64(1), nil, 0, nil, 11, bufAliasPayloadB, 1},
		},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[set]) != 2 {
		t.Fatalf("section items: got %d, want 2", len(res.items[set]))
	}

	first := res.items[set][0].(bufAliasChild)
	second := res.items[set][1].(bufAliasChild)
	if string(first.Blob) != string(bufAliasPayloadA) {
		t.Fatalf(
			"first child model JSON = %q, want %q: under a restricted ColumnPlan the section buffer "+
				"is not recreated per row, so the second row's Scan overwrote the first model's backing array",
			first.Blob, bufAliasPayloadA,
		)
	}
	if string(second.Blob) != string(bufAliasPayloadB) {
		t.Fatalf("second child model JSON = %q, want %q", second.Blob, bufAliasPayloadB)
	}
}
