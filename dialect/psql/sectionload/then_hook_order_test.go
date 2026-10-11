// These tests pin the ordering between Then(...) boundaries and the parent's
// AfterQueryHook. sectionload.All must finish running the Then(...)
// boundaries before the parent's AfterQueryHook; otherwise an
// AfterSelectHook registered on the parent model sees an incomplete graph in
// which the relation fields (R) that Then(...) should have loaded are not
// yet stitched (bob's exec.go runs loaders before after-select hooks).
//
// No database is used. A minimal fake Executor intercepts QueryContext and
// returns hard-coded rows (bob.Executor is defined in bob's exec.go as
// scan.Queryer (QueryContext) plus ExecContext).
package sectionload_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/scan"

	sectionload "github.com/stephenafamo/bob/dialect/psql/sectionload"
)

// --- Minimal test-only models -------------------------------------------------
//
// These are hand-written in the same shape as generated code (a table's
// parts and an edge definition), without a DB schema. The real thing is a
// bob-generated model, but a minimal struct with just an R field and an
// AfterQueryHook is enough here.

type fakeChild struct {
	ID int64
}

type fakeParent struct {
	ID      int64
	ChildID int64
	R       fakeParentRels
}

type fakeParentRels struct {
	Child *fakeChild
}

// fakeParentSlice implements bob.HookableType, mirroring how generated
// code's <Table>Slice types carry AfterQueryHook.
type fakeParentSlice []*fakeParent

// hookObservations records, per element of the slice the hook was called
// with, whether R.Child was already stitched at hook-fire time. This is
// the thing Step 2 of the brief asks the test to check.
var hookObservations []bool

func (s fakeParentSlice) AfterQueryHook(_ context.Context, _ bob.Executor, _ bob.QueryType) error {
	hookObservations = make([]bool, len(s))
	for i, p := range s {
		hookObservations[i] = p.R.Child != nil
	}
	return nil
}

// --- TableParts / Edge (equivalent to generated code) --------------------------

type fakeParentBuf struct {
	ID      int64
	ChildID sql.NullInt64
}

func (b *fakeParentBuf) TargetsFor(cols []sectionload.Column) []any { return []any{&b.ID, &b.ChildID} }

func (b *fakeParentBuf) BuildFor(modelCols []sectionload.Column) *fakeParent {
	p := &fakeParent{ID: b.ID}
	if b.ChildID.Valid {
		p.ChildID = b.ChildID.Int64
	}
	return p
}

type fakeChildBuf struct {
	ID int64
}

func (b *fakeChildBuf) TargetsFor(cols []sectionload.Column) []any { return []any{&b.ID} }
func (b *fakeChildBuf) BuildFor(modelCols []sectionload.Column) *fakeChild {
	return &fakeChild{ID: b.ID}
}

var fakeParentParts = sectionload.TableParts[*fakeParent]{
	Schema: "",
	Table:  "fake_parents",
	Columns: []sectionload.Column{
		{Name: "id", DBType: "bigint"},
		{Name: "child_id", DBType: "bigint"},
	},
	NewBuf:  func() sectionload.Buf[*fakeParent] { return &fakeParentBuf{} },
	AsSlice: func(v []*fakeParent) any { return fakeParentSlice(v) },
}

var fakeChildParts = sectionload.TableParts[*fakeChild]{
	Schema: "",
	Table:  "fake_children",
	Columns: []sectionload.Column{
		{Name: "id", DBType: "bigint"},
	},
	NewBuf: func() sectionload.Buf[*fakeChild] { return &fakeChildBuf{} },
}

// fakeParentChildEdge is a to-one edge, equivalent in shape to
// comments.bob.go's commentSections.Post() edge.
var fakeParentChildEdge = sectionload.Edge[*fakeParent, *fakeChild]{
	RelName:  "Child",
	FromCols: []string{"child_id"},
	ToCols:   []string{"id"},
	ToMany:   false,
	Child:    fakeChildParts,
	ParentKey: func(p *fakeParent) []any {
		return []any{p.ChildID}
	},
	ChildKey: func(c *fakeChild) []any {
		return []any{c.ID}
	},
	Stitch: func(p *fakeParent, children []*fakeChild) error {
		if len(children) == 0 {
			return nil
		}
		p.R.Child = children[0]
		return nil
	},
}

// --- Fake Executor -------------------------------------------------------
//
// bob.Executor requires scan.Queryer (QueryContext(ctx, query string,
// args ...any) (scan.Rows, error)) and ExecContext. The order of the queries
// sectionload.All runs is fixed:
//
//	1st: the main island (parent table only; Then(...) is not included)
//	2nd: the Then boundary's query issued by runThen (child table)
//
// The fake decides which query it is from the call count and returns the
// matching fixed rows. It does not inspect the SQL text (these tests care
// about execution order, not SQL generation).
type fakeExecutor struct {
	calls   int
	queries []string
}

func (f *fakeExecutor) QueryContext(_ context.Context, query string, _ ...any) (scan.Rows, error) {
	f.calls++
	f.queries = append(f.queries, query)
	switch f.calls {
	case 1:
		// Main island: one fake_parents row (id=1, child_id=42).
		return newFakeRows(
			[]string{"_sl_set", "_sl_ord", "id", "child_id"},
			[][]any{
				{int64(0), sql.NullInt64{Int64: 1, Valid: true}, int64(1), sql.NullInt64{Int64: 42, Valid: true}},
			},
		), nil
	case 2:
		// Then boundary: one fake_children row (id=42).
		return newFakeRows(
			[]string{"_sl_set", "_sl_ord", "id"},
			[][]any{
				{int64(0), sql.NullInt64{}, int64(42)},
			},
		), nil
	default:
		return nil, fmt.Errorf("fakeExecutor: unexpected QueryContext call #%d: %s", f.calls, query)
	}
}

func (f *fakeExecutor) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	return nil, fmt.Errorf("fakeExecutor: unexpected ExecContext call: %s", query)
}

// fakeRows is the minimal scan.Rows (interfaces.go, scan@v0.9.0) implementation:
// Scan/Columns/Next/Close/Err over a hardcoded [][]any row set.
type fakeRows struct {
	cols []string
	data [][]any
	pos  int
}

func newFakeRows(cols []string, data [][]any) *fakeRows {
	return &fakeRows{cols: cols, data: data, pos: -1}
}

func (r *fakeRows) Columns() ([]string, error) { return r.cols, nil }

func (r *fakeRows) Next() bool {
	r.pos++
	return r.pos < len(r.data)
}

func (r *fakeRows) Scan(dest ...any) error {
	row := r.data[r.pos]
	if len(dest) != len(row) {
		return fmt.Errorf("fakeRows: dest count %d != row width %d", len(dest), len(row))
	}
	for i, d := range dest {
		switch d := d.(type) {
		case *int64:
			v, ok := row[i].(int64)
			if !ok {
				return fmt.Errorf("fakeRows: col %d: want int64, got %T", i, row[i])
			}
			*d = v
		case *sql.NullInt64:
			v, ok := row[i].(sql.NullInt64)
			if !ok {
				return fmt.Errorf("fakeRows: col %d: want sql.NullInt64, got %T", i, row[i])
			}
			*d = v
		default:
			return fmt.Errorf("fakeRows: unsupported scan dest type %T at col %d", d, i)
		}
	}
	return nil
}

func (r *fakeRows) Close() error { return nil }
func (r *fakeRows) Err() error   { return nil }

// --- Tests --------------------------------------------------------------------

// TestThenBoundaryBeforeParentHook asserts that by the time the parent's
// AfterQueryHook fires, a Then(...) boundary's relation has already been
// stitched into R. If the parent hook ran before the Then boundaries, this
// would fail: the hook would see R.Child == nil.
func TestThenBoundaryBeforeParentHook(t *testing.T) {
	hookObservations = nil
	ctx := context.Background()
	exec := &fakeExecutor{}

	parentQuery := psql.RawQuery(`SELECT id, child_id FROM fake_parents`)

	got, err := sectionload.All(
		ctx,
		exec,
		parentQuery,
		fakeParentParts,
		sectionload.Then(sectionload.NewNode(fakeParentChildEdge)),
	)
	if err != nil {
		t.Fatalf("sectionload.All: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected parent count: got %d want 1", len(got))
	}
	if exec.calls != 2 {
		t.Fatalf("unexpected QueryContext call count: got %d want 2 (main island + Then boundary)", exec.calls)
	}
	if len(hookObservations) != 1 {
		t.Fatalf("unexpected number of AfterQueryHook observations: got %d want 1 (the hook may not have fired at all)", len(hookObservations))
	}
	if !hookObservations[0] {
		t.Fatalf("R.Child of the Then(...) boundary was unset when AfterQueryHook ran: " +
			"the parent hook ran before the Then boundaries")
	}
	if got[0].R.Child == nil || got[0].R.Child.ID != 42 {
		t.Fatalf("R.Child in the final result was not stitched as expected: %+v", got[0].R)
	}
}
