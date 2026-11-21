package bob

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/stephenafamo/scan"
)

// This file characterizes and then locks down the fix for P-18: Allx and
// QueryStmt.All disagreed on what to do when the slice type is not
// HookableType but the single element type is. Allx rejects the query
// before it runs (ErrHookableTypeMismatch); QueryStmt.All used to silently
// fall back to calling AfterQueryHook on each element. This file adds a
// shared table test covering both entry points, backed by DB-free fakes.

// ---- fake scan.Rows ----

type p18FakeRows struct {
	values []int64
	idx    int
}

func (r *p18FakeRows) Columns() ([]string, error) { return []string{"v"}, nil }

func (r *p18FakeRows) Next() bool {
	r.idx++
	return r.idx <= len(r.values)
}

func (r *p18FakeRows) Scan(dest ...any) error {
	reflect.ValueOf(dest[0]).Elem().SetInt(r.values[r.idx-1])
	return nil
}

func (r *p18FakeRows) Close() error { return nil }
func (r *p18FakeRows) Err() error   { return nil }

// ---- fake Executor, for the Allx (non-prepared) path ----

type p18FakeExecutor struct {
	rowValues  []int64
	queryCalls int
}

func (e *p18FakeExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	e.queryCalls++
	return &p18FakeRows{values: e.rowValues}, nil
}

func (e *p18FakeExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

// ---- fake PreparedExecutor + Preparer, for the QueryStmt.All (prepared) path ----

type p18FakeStmt struct {
	rowValues  []int64
	queryCalls int
}

func (s *p18FakeStmt) QueryContext(ctx context.Context, args ...any) (scan.Rows, error) {
	s.queryCalls++
	return &p18FakeRows{values: s.rowValues}, nil
}

func (s *p18FakeStmt) ExecContext(ctx context.Context, args ...any) (sql.Result, error) {
	return nil, nil
}

func (s *p18FakeStmt) Close() error { return nil }

type p18FakePreparer struct {
	stmt *p18FakeStmt
}

func (p *p18FakePreparer) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return nil, nil // unused: these tests attach no loaders
}

func (p *p18FakePreparer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (p *p18FakePreparer) PrepareContext(ctx context.Context, query string) (*p18FakeStmt, error) {
	return p.stmt, nil
}

// ---- fake bob.Query: a fixed 1-column SELECT ----

type p18FakeQuery struct{}

func (p18FakeQuery) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	_, err := w.WriteString("SELECT 1")
	return nil, err
}

func (p18FakeQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	_, err := w.WriteString("SELECT 1")
	return nil, err
}

func (p18FakeQuery) Type() QueryType { return QueryTypeSelect }

// ---- element/slice type combinations under test ----

// p18Elem implements HookableType. p18PlainSlice ([]p18Elem) does not.
// This is the mismatch combination the issue is about: slice not hookable,
// but the single/element type is.
type p18Elem int

type p18HookRecord struct {
	count     int
	lastQType QueryType
}

var p18ElemHookCalls p18HookRecord

func (p18Elem) AfterQueryHook(ctx context.Context, exec Executor, qt QueryType) error {
	p18ElemHookCalls.count++
	p18ElemHookCalls.lastQType = qt
	return nil
}

type p18PlainSlice []p18Elem // no AfterQueryHook: the mismatch combination

var (
	p18SliceHookCalls     int
	p18SliceHookLastQType QueryType
)

type p18HookableSlice []p18Elem // both slice and element are hookable

func (p18HookableSlice) AfterQueryHook(ctx context.Context, exec Executor, qt QueryType) error {
	p18SliceHookCalls++
	p18SliceHookLastQType = qt
	return nil
}

type (
	p18PlainElem      int // neither the element nor the slice is hookable
	p18PlainElemSlice []p18PlainElem
)

func p18ResetCounters() {
	p18ElemHookCalls = p18HookRecord{}
	p18SliceHookCalls = 0
	p18SliceHookLastQType = QueryTypeUnknown
}

// p18RunAllx runs the Allx entry point for the given T/Ts combination and
// reports the executor's query-call count alongside the result. The error
// is returned last (not the query-call count) to satisfy the repo's
// error-last convention (staticcheck ST1008).
func p18RunAllx[Tr Transformer[T, Ts], T, Ts any](t *testing.T, rowValues []int64) (Ts, int, error) {
	t.Helper()
	exec := &p18FakeExecutor{rowValues: rowValues}
	result, err := Allx[Tr, T, Ts](context.Background(), exec, p18FakeQuery{}, p18ScanMapper[T]())
	return result, exec.queryCalls, err
}

// p18RunPreparedAll runs PrepareQueryx(...).All for the given T/Ts
// combination and reports the underlying stmt's query-call count alongside
// the result.
func p18RunPreparedAll[T any, Ts ~[]T](t *testing.T, rowValues []int64) (Ts, int, error) {
	t.Helper()
	stmt := &p18FakeStmt{rowValues: rowValues}
	preparer := &p18FakePreparer{stmt: stmt}

	qs, err := PrepareQueryx[struct{}, *p18FakeStmt, T, Ts](context.Background(), preparer, p18FakeQuery{}, p18ScanMapper[T]())
	if err != nil {
		t.Fatalf("PrepareQueryx: %v", err)
	}

	result, err := qs.All(context.Background(), struct{}{})
	return result, stmt.queryCalls, err
}

// p18ScanMapper returns scan.SingleColumnMapper[T], typed as scan.Mapper[T]
// so it can be passed to both Allx and PrepareQueryx without repeating the
// type parameter at every call site.
func p18ScanMapper[T any]() scan.Mapper[T] {
	return scan.SingleColumnMapper[T]
}

func TestHookableTypeMismatch(t *testing.T) {
	t.Run("case1_Allx_mismatch_rejects_before_executing", func(t *testing.T) {
		p18ResetCounters()

		_, queryCalls, err := p18RunAllx[SliceTransformer[p18Elem, p18PlainSlice], p18Elem, p18PlainSlice](t, []int64{1, 2})

		if !errors.Is(err, ErrHookableTypeMismatch) {
			t.Fatalf("expected ErrHookableTypeMismatch, got %v", err)
		}
		if queryCalls != 0 {
			t.Fatalf("expected the query to never run (0 QueryContext calls), got %d", queryCalls)
		}
		if p18ElemHookCalls.count != 0 {
			t.Fatalf("expected 0 element hook calls, got %d", p18ElemHookCalls.count)
		}
	})

	t.Run("case2_PrepareQueryxAll_mismatch", func(t *testing.T) {
		p18ResetCounters()

		_, queryCalls, err := p18RunPreparedAll[p18Elem, p18PlainSlice](t, []int64{1, 2})

		// Post-fix contract: QueryStmt.All must mirror Allx and reject
		// before executing the query.
		if !errors.Is(err, ErrHookableTypeMismatch) {
			t.Fatalf("expected ErrHookableTypeMismatch, got %v", err)
		}
		if queryCalls != 0 {
			t.Fatalf("expected the query to never run (0 QueryContext calls on the stmt), got %d", queryCalls)
		}
		if p18ElemHookCalls.count != 0 {
			t.Fatalf("expected 0 element hook calls (no fallback), got %d", p18ElemHookCalls.count)
		}
		if p18SliceHookCalls != 0 {
			t.Fatalf("expected 0 slice hook calls, got %d", p18SliceHookCalls)
		}
	})

	t.Run("case3_both_hookable_slice_hook_fires_once_both_paths", func(t *testing.T) {
		for _, path := range []string{"Allx", "PrepareQueryx.All"} {
			t.Run(path, func(t *testing.T) {
				p18ResetCounters()

				var err error
				var queryCalls int
				switch path {
				case "Allx":
					_, queryCalls, err = p18RunAllx[SliceTransformer[p18Elem, p18HookableSlice], p18Elem, p18HookableSlice](t, []int64{1, 2})
				case "PrepareQueryx.All":
					_, queryCalls, err = p18RunPreparedAll[p18Elem, p18HookableSlice](t, []int64{1, 2})
				}

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if queryCalls != 1 {
					t.Fatalf("expected the query to run exactly once, got %d", queryCalls)
				}
				if p18SliceHookCalls != 1 {
					t.Fatalf("expected the slice hook to fire exactly once, got %d", p18SliceHookCalls)
				}
				if p18ElemHookCalls.count != 0 {
					t.Fatalf("expected 0 element hook calls (slice hook takes priority), got %d", p18ElemHookCalls.count)
				}
			})
		}
	})

	t.Run("case4_neither_hookable_no_hook_calls_both_paths", func(t *testing.T) {
		for _, path := range []string{"Allx", "PrepareQueryx.All"} {
			t.Run(path, func(t *testing.T) {
				p18ResetCounters()

				var err error
				var queryCalls int
				switch path {
				case "Allx":
					_, queryCalls, err = p18RunAllx[SliceTransformer[p18PlainElem, p18PlainElemSlice], p18PlainElem, p18PlainElemSlice](t, []int64{1, 2})
				case "PrepareQueryx.All":
					_, queryCalls, err = p18RunPreparedAll[p18PlainElem, p18PlainElemSlice](t, []int64{1, 2})
				}

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if queryCalls != 1 {
					t.Fatalf("expected the query to run exactly once, got %d", queryCalls)
				}
				if p18SliceHookCalls != 0 || p18ElemHookCalls.count != 0 {
					t.Fatalf("expected 0 hook calls of any kind, got slice=%d element=%d", p18SliceHookCalls, p18ElemHookCalls.count)
				}
			})
		}
	})

	t.Run("case5_prepared_path_passes_QueryType_through_to_the_hook", func(t *testing.T) {
		// Same combination as case 3 (both hookable), prepared path only,
		// checking what QueryType reaches AfterQueryHook. Related to P-19
		// (Clone() dropping QueryType): PrepareQueryx stores queryType from
		// q.Type() directly at prepare time (stmt.go), it does not go
		// through BaseQuery.Clone(), so this does not exercise P-19's bug.
		p18ResetCounters()

		_, _, err := p18RunPreparedAll[p18Elem, p18HookableSlice](t, []int64{1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p18SliceHookCalls != 1 {
			t.Fatalf("expected the slice hook to fire exactly once, got %d", p18SliceHookCalls)
		}
		if p18SliceHookLastQType != QueryTypeSelect {
			t.Fatalf("expected QueryType passed to hook to be QueryTypeSelect, got %v", p18SliceHookLastQType)
		}
	})
}
