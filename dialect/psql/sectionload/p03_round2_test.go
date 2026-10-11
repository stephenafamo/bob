package sectionload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/scan"
)

// TestPlanIsland_ThenBeforeSectionDoesNotCreateSetGap is the direct
// regression test for P-03 round-2 Critical 1: a Then(...) declared
// BEFORE a normal sibling used to consume a section set number, leaving
// the sibling with set=2 while execIsland sizes items as
// len(sections)+1 == 2 — an out-of-range panic.
func TestPlanIsland_ThenBeforeSectionDoesNotCreateSetGap(t *testing.T) {
	tree := []Node{
		Then(NewNode(p28RootMidEdge())),
		NewNode(p28RootMidEdge()),
		Then(NewNode(p28RootMidEdge())),
	}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	if len(plan.sections) != 1 || plan.sections[0].set != 1 {
		t.Fatalf("sections must stay dense and 1-based regardless of Then order; got %+v", plan.sections)
	}
	if len(plan.thens) != 2 {
		t.Fatalf("expected 2 then boundaries, got %+v", plan.thens)
	}
	if plan.thens[0].id != -1 || plan.thens[1].id != -2 {
		t.Fatalf("then IDs must live in their own negative space (-1, -2); got %d, %d",
			plan.thens[0].id, plan.thens[1].id)
	}
}

// TestExecIsland_ThenBeforeSectionDoesNotPanic drives execIsland with a
// row belonging to the normal section declared AFTER a Then boundary.
// Before the fix this indexes items[2] with len(items)==2 and panics.
func TestExecIsland_ThenBeforeSectionDoesNotPanic(t *testing.T) {
	tree := []Node{
		Then(NewNode(p28RootMidEdge())),
		NewNode(p28RootMidEdge()),
	}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 {
		t.Fatalf("expected 1 section, got %+v", plan.sections)
	}
	set := plan.sections[0].set

	root := partsAdapter[p28Root]{TableParts[p28Root]{
		Table:   "p28_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
	}}

	cols := expectedColumns(root, plan)
	rows := &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id, mid.id, mid.root_id, mid.other_key
			{int64(0), int64(1), 100, 0, 0, 0},
			{int64(set), nil, 0, 200, 100, 999},
		}},
	}
	exec := p28FakeExecutor{rows: rows}

	res, err := execIsland(
		context.Background(), exec, psql.RawQuery("SELECT 1"),
		root, plan, nil,
	)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[1]) != 1 {
		t.Fatalf("section row must land in items[1]; got items[1] len=%d", len(res.items[1]))
	}
}

// p03r2SeqExecutor returns a different canned row set per QueryContext
// call: call 1 feeds All's island query, call 2 feeds runThen's child
// island query.
type p03r2SeqExecutor struct {
	rowSets []scan.Rows
	calls   int
}

func (e *p03r2SeqExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	e.calls++
	return e.rowSets[e.calls-1], nil
}

func (e *p03r2SeqExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("p03r2SeqExecutor: ExecContext not implemented")
}

// TestAll_RootThenKeysFollowOrdinalRestoration is the direct regression
// test for P-03 round-2 Critical 2. Scan order deliberately differs from
// _sl_ord order: pre-fix, All reorders items[0] only, so parents[i] gets
// thenKeys[i] of a DIFFERENT parent and runThen stitches the wrong
// children (row counts match, so no guard trips).
func TestAll_RootThenKeysFollowOrdinalRestoration(t *testing.T) {
	stitched := map[int][]int{} // root.ID -> child IDs

	edge := Edge[p28Root, p28Mid]{
		RelName:  "Mid",
		FromCols: []string{"id"},
		ToCols:   []string{"root_id"},
		ToMany:   true,
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
		Stitch: func(p p28Root, cs []p28Mid) error {
			ids := make([]int, 0, len(cs))
			for _, c := range cs {
				ids = append(ids, c.ID)
			}
			stitched[p.ID] = ids
			return nil
		},
	}

	parts := TableParts[p28Root]{
		Table:   "p28_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
	}
	tree := []Node{Then(NewNode(edge))}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := partsAdapter[p28Root]{parts}

	// Call 1 (island): scan order [id=20 ord=2, id=10 ord=1] — reversed
	// vs _sl_ord. No normal sections, so layout is _sl_set,_sl_ord,root.id.
	islandRows := &p28ColsRows{
		cols: expectedColumns(root, plan),
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			{int64(0), int64(2), 20},
			{int64(0), int64(1), 10},
		}},
	}

	// Call 2 (runThen child island): children in their own island layout
	// (_sl_set,_sl_ord,mid.id,mid.root_id,mid.other_key). Child 111
	// belongs to root 10, child 222 to root 20.
	childRoot := edgeRootParts{edge.child(), edge}
	childPlan, err := planIsland(nil)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	childRows := &p28ColsRows{
		cols: expectedColumns(childRoot, childPlan),
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			{int64(0), int64(1), 111, 10, 0},
			{int64(0), int64(2), 222, 20, 0},
		}},
	}

	exec := &p03r2SeqExecutor{rowSets: []scan.Rows{islandRows, childRows}}
	out, err := All(context.Background(), exec, psql.RawQuery("SELECT 1"), parts, tree...)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(out) != 2 || out[0].ID != 10 || out[1].ID != 20 {
		t.Fatalf("parents must come back in _sl_ord order [10 20]; got %+v", out)
	}

	want := map[int][]int{10: {111}, 20: {222}}
	for id, wantIDs := range want {
		gotIDs := stitched[id]
		if len(gotIDs) != len(wantIDs) || (len(gotIDs) > 0 && gotIDs[0] != wantIDs[0]) {
			t.Fatalf("root %d got children %v, want %v (thenKeys not reordered with parents?)", id, gotIDs, wantIDs)
		}
	}
}

// p03r2HookedQuery wraps a bob.Query and overrides RunHooks so tests can
// observe whether/when it ran, and force it to return an error, without a
// real hookable query type.
//
// P-13 (2026-08-20): All() now reads bob PreloadCount declarations
// ([0.6] in run.go) BEFORE validatePlan runs — before this addition, a
// query type declaredCountPreloads cannot introspect only ever surfaced
// as *UncheckableParentQueryError inside [1.6]'s
// installBuildTimeMapperModCheck, which TestAll_ValidatePlanRunsAfter
// ParentQueryHook never reached (validatePlan itself failed first with
// the MissingKeyFromBufError this test wants). declaredCountPreloads
// uses the same psqlSelectModdable + no-op-mod trick as
// installBuildTimeMapperModCheck to reach the concrete
// *dialect.SelectQuery, but bob.Query (the embedded field's static
// type) does not itself declare Apply(...bob.Mod[*dialect.SelectQuery]),
// so that method is never promoted from an embedded interface value —
// this wrapper must expose it explicitly, delegating to whatever
// concrete Query it wraps.
type p03r2HookedQuery struct {
	bob.Query
	hookRan *bool
	hookErr error
}

func (q p03r2HookedQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	*q.hookRan = true
	return ctx, q.hookErr
}

func (q p03r2HookedQuery) Apply(mods ...bob.Mod[*dialect.SelectQuery]) {
	if a, ok := q.Query.(interface {
		Apply(...bob.Mod[*dialect.SelectQuery])
	}); ok {
		a.Apply(mods...)
	}
}

// TestAll_ValidatePlanRunsAfterParentQueryHook is the direct regression
// test for P-03 round-2 Minor 2: validatePlan used to run BEFORE the
// parent query's hooks ([0] in run.go), breaking bob's hook-first
// contract (P-06 requires tenancy-condition hooks to always run, even
// when the declared tree itself is invalid). It must run AFTER the hooks
// but still before any SQL — this test wires a plan that is guaranteed to
// fail validatePlan (a restricted edge with a nil ChildKeyFromBuf) and
// checks that the parent hook ran, and that the executor was never
// reached, before the (expected) MissingKeyFromBufError comes back.
func TestAll_ValidatePlanRunsAfterParentQueryHook(t *testing.T) {
	ran := false
	parent := p03r2HookedQuery{Query: psql.Select(), hookRan: &ran}
	badEdge := p28RootMidEdge()
	badEdge.ColumnPlan = ColumnPlan{Model: []Column{{Name: "id", DBType: "int4"}}}
	// ChildKeyFromBuf left nil deliberately, to induce MissingKeyFromBufError.
	exec := &p03r2SeqExecutor{} // QueryContext reached => index out of range

	_, err := All(context.Background(), exec, parent,
		TableParts[p28Root]{
			Table:   "p28_root",
			Columns: []Column{{Name: "id", DBType: "int4"}},
			NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
		},
		NewNode(badEdge))
	var want *MissingKeyFromBufError
	if !errors.As(err, &want) {
		t.Fatalf("expected MissingKeyFromBufError, got %v", err)
	}
	if !ran {
		t.Fatal("parent hook must run BEFORE plan validation (bob's hook-first contract)")
	}
	if exec.calls != 0 {
		t.Fatalf("executor must not be reached, got %d calls", exec.calls)
	}
}

// TestAll_HookErrorPrecedesValidationError checks the ordering the other
// way: even with a plan that would fail validatePlan, a hook error must
// win — the caller should see the hook's own error, not have it masked by
// a validation error surfacing first.
func TestAll_HookErrorPrecedesValidationError(t *testing.T) {
	ran := false
	sentinel := errors.New("hook sentinel")
	parent := p03r2HookedQuery{Query: psql.RawQuery("SELECT 1"), hookRan: &ran, hookErr: sentinel}
	badEdge := p28RootMidEdge()
	badEdge.ColumnPlan = ColumnPlan{Model: []Column{{Name: "id", DBType: "int4"}}}

	_, err := All(context.Background(), &p03r2SeqExecutor{}, parent,
		TableParts[p28Root]{
			Table:   "p28_root",
			Columns: []Column{{Name: "id", DBType: "int4"}},
			NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
		},
		NewNode(badEdge))
	if !errors.Is(err, sentinel) {
		t.Fatalf("hook error must win over validation error, got %v", err)
	}
}
