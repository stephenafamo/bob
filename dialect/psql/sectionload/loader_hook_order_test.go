// These tests pin the ordering between a loader attached directly to the
// parent query and a registered after-select hook.
//
// sectionload.All must finish running loaders attached to the parent query
// before the parent's AfterQueryHook runs (bob's exec.go runs loaders before
// after-select hooks). Otherwise the hook cannot see fields written by the
// loader, and a loader error cannot stop the hook from running.
//
// No database is used. A minimal fake Executor intercepts QueryContext and
// returns hard-coded rows (bob.Executor is defined in bob's exec.go as
// scan.Queryer (QueryContext) plus ExecContext).
package sectionload_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"

	sectionload "github.com/stephenafamo/bob/dialect/psql/sectionload"
)

// --- Minimal test-only models -------------------------------------------------

type loaderHookFakeParent struct {
	ID int64
	// LoadedAt is written by the parent-attached loader. If the hook can
	// read "loaded" here, the loader ran before the hook.
	LoadedAt string
}

// loaderHookFakeParentSlice implements bob.HookableType, mirroring how generated
// code's <Table>Slice types carry AfterQueryHook.
type loaderHookFakeParentSlice []*loaderHookFakeParent

var (
	// loaderHookAfterSelectHooks mirrors generated code's <Table>.AfterSelectHooks
	// registry (e.g. Posts.AfterSelectHooks, posts.bob.go:1053-1060):
	// AfterQueryHook dispatches to it via bob.Hooks.RunHooks instead of
	// containing hook logic directly, so this test exercises the same
	// AppendHooks/RunHooks path real registered hooks use.
	loaderHookAfterSelectHooks *bob.Hooks[loaderHookFakeParentSlice, bob.SkipModelHooksKey]
	loaderHookTrace            []string
	loaderHookSawValue         string
	errLoaderHook              error
)

func (s loaderHookFakeParentSlice) AfterQueryHook(
	ctx context.Context,
	exec bob.Executor,
	queryType bob.QueryType,
) error {
	var err error
	switch queryType {
	case bob.QueryTypeSelect:
		_, err = loaderHookAfterSelectHooks.RunHooks(ctx, exec, s)
	}
	return err
}

// loaderHookRegisterAfterSelectHook installs a fresh AfterSelectHooks registration,
// mirroring how generated code registers hooks via AppendHooks. Call this
// before sectionload.All in each test.
func loaderHookRegisterAfterSelectHook() {
	loaderHookAfterSelectHooks = new(bob.Hooks[loaderHookFakeParentSlice, bob.SkipModelHooksKey])
	loaderHookAfterSelectHooks.AppendHooks(
		func(
			ctx context.Context,
			_ bob.Executor,
			parents loaderHookFakeParentSlice,
		) (context.Context, error) {
			loaderHookTrace = append(loaderHookTrace, "hook")
			if len(parents) > 0 {
				loaderHookSawValue = parents[0].LoadedAt
			}
			return ctx, errLoaderHook
		},
	)
}

// --- TableParts (equivalent to generated code) --------------------------------

type loaderHookFakeParentBuf struct {
	ID int64
}

func (b *loaderHookFakeParentBuf) TargetsFor(cols []sectionload.Column) []any { return []any{&b.ID} }

func (b *loaderHookFakeParentBuf) BuildFor(modelCols []sectionload.Column) *loaderHookFakeParent {
	return &loaderHookFakeParent{ID: b.ID}
}

var loaderHookFakeParentParts = sectionload.TableParts[*loaderHookFakeParent]{
	Schema: "",
	Table:  "lh_fake_parents",
	Columns: []sectionload.Column{
		{Name: "id", DBType: "bigint"},
	},
	NewBuf:  func() sectionload.Buf[*loaderHookFakeParent] { return &loaderHookFakeParentBuf{} },
	AsSlice: func(v []*loaderHookFakeParent) any { return loaderHookFakeParentSlice(v) },
}

// --- Wrapper that attaches a loader directly to the parent query ---------------
//
// A wrapper that satisfies bob.Loadable and returns the test loader. It
// embeds the bob.BaseQuery[*dialect.SelectQuery] returned by psql.Select. It
// must not embed psql.RawQuery's bob.BaseQuery[expr.Clause]: the build-time
// MapperMod check (installBuildTimeMapperModCheck in run.go) treats only
// concrete types exposing Apply(...bob.Mod[*dialect.SelectQuery]) as safe
// and rejects wrappers whose dynamic type does not match with
// UncheckableParentQueryError, and a named struct embedding
// bob.BaseQuery[expr.Clause] does not match that type check. GetLoaders is
// overridden on this wrapper to return the loader attached to the parent
// query explicitly.
type loaderHookLoadableQuery struct {
	bob.BaseQuery[*dialect.SelectQuery]
	loaders []bob.Loader
}

func (q loaderHookLoadableQuery) GetLoaders() []bob.Loader { return q.loaders }

var (
	_ bob.Loadable = psql.RawQuery("SELECT 1")
	_ bob.Loadable = loaderHookLoadableQuery{}
)

func loaderHookNewLoadableParentQuery(loader bob.Loader) loaderHookLoadableQuery {
	return loaderHookLoadableQuery{
		BaseQuery: psql.Select(sm.From(psql.Raw("lh_fake_parents")), sm.Columns(psql.Raw("id"))),
		loaders:   []bob.Loader{loader},
	}
}

// --- Fake Executor -------------------------------------------------------

type loaderHookFakeExecutor struct {
	calls int
}

func (f *loaderHookFakeExecutor) QueryContext(_ context.Context, query string, _ ...any) (scan.Rows, error) {
	f.calls++
	if f.calls != 1 {
		return nil, fmt.Errorf("loaderHookFakeExecutor: unexpected QueryContext call #%d: %s", f.calls, query)
	}
	// The single row of the main island. The column layout is fixed (_sl_set, _sl_ord, <root cols>).
	return loaderHookNewFakeRows(
		[]string{"_sl_set", "_sl_ord", "id"},
		[][]any{
			{int64(0), sql.NullInt64{Int64: 1, Valid: true}, int64(1)},
		},
	), nil
}

func (f *loaderHookFakeExecutor) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	return nil, fmt.Errorf("loaderHookFakeExecutor: unexpected ExecContext call: %s", query)
}

// loaderHookFakeRows is the minimal scan.Rows (interfaces.go, scan@v0.9.0)
// implementation: Scan/Columns/Next/Close/Err over a hardcoded [][]any row
// set.
type loaderHookFakeRows struct {
	cols []string
	data [][]any
	pos  int
}

func loaderHookNewFakeRows(cols []string, data [][]any) *loaderHookFakeRows {
	return &loaderHookFakeRows{cols: cols, data: data, pos: -1}
}

func (r *loaderHookFakeRows) Columns() ([]string, error) { return r.cols, nil }

func (r *loaderHookFakeRows) Next() bool {
	r.pos++
	return r.pos < len(r.data)
}

func (r *loaderHookFakeRows) Scan(dest ...any) error {
	row := r.data[r.pos]
	if len(dest) != len(row) {
		return fmt.Errorf("loaderHookFakeRows: dest count %d != row width %d", len(dest), len(row))
	}
	for i, d := range dest {
		switch d := d.(type) {
		case *int64:
			v, ok := row[i].(int64)
			if !ok {
				return fmt.Errorf("loaderHookFakeRows: col %d: want int64, got %T", i, row[i])
			}
			*d = v
		case *sql.NullInt64:
			v, ok := row[i].(sql.NullInt64)
			if !ok {
				return fmt.Errorf("loaderHookFakeRows: col %d: want sql.NullInt64, got %T", i, row[i])
			}
			*d = v
		default:
			return fmt.Errorf("loaderHookFakeRows: unsupported scan dest type %T at col %d", d, i)
		}
	}
	return nil
}

func (r *loaderHookFakeRows) Close() error { return nil }
func (r *loaderHookFakeRows) Err() error   { return nil }

// --- Tests --------------------------------------------------------------------

// TestParentLoaderRunsBeforeParentHook asserts that by the time the
// parent's AfterQueryHook fires, a loader attached directly to the parent
// query has already run, and that the hook observes the loader's side
// effect on the shared model slice.
func TestParentLoaderRunsBeforeParentHook(t *testing.T) {
	loaderHookTrace = nil
	loaderHookSawValue = ""
	errLoaderHook = nil
	loaderHookRegisterAfterSelectHook()
	ctx := context.Background()
	exec := &loaderHookFakeExecutor{}

	loader := bob.LoaderFunc(func(_ context.Context, _ bob.Executor, retrieved any) error {
		loaderHookTrace = append(loaderHookTrace, "loader")
		parents, ok := retrieved.(loaderHookFakeParentSlice)
		if !ok {
			return fmt.Errorf("loader: unexpected retrieved type %T", retrieved)
		}
		for _, p := range parents {
			p.LoadedAt = "loaded"
		}
		return nil
	})

	got, err := sectionload.All(
		ctx,
		exec,
		loaderHookNewLoadableParentQuery(loader),
		loaderHookFakeParentParts,
	)
	if err != nil {
		t.Fatalf("sectionload.All: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected parent count: got %d want 1", len(got))
	}
	if want := []string{"loader", "hook"}; len(loaderHookTrace) != 2 || loaderHookTrace[0] != want[0] || loaderHookTrace[1] != want[1] {
		t.Fatalf("unexpected loader/hook order: got %v want %v", loaderHookTrace, want)
	}
	if loaderHookSawValue != "loaded" {
		t.Fatalf("the hook did not see the loader's write (LoadedAt) when AfterQueryHook ran: got %q", loaderHookSawValue)
	}
	if got[0].LoadedAt != "loaded" {
		t.Fatalf("the loader's write is missing from the final result: %+v", got[0])
	}
}

// TestParentLoaderErrorSuppressesParentHook asserts that when the
// parent-attached loader returns an error, the parent's AfterQueryHook
// does not run at all — matching bob's own short-circuit behavior in exec.go
// (a loader error short-circuits before the hooks).
func TestParentLoaderErrorSuppressesParentHook(t *testing.T) {
	loaderHookTrace = nil
	errLoaderHook = nil
	loaderHookRegisterAfterSelectHook()
	ctx := context.Background()
	exec := &loaderHookFakeExecutor{}
	wantErr := errors.New("loader boom")

	loader := bob.LoaderFunc(func(_ context.Context, _ bob.Executor, _ any) error {
		loaderHookTrace = append(loaderHookTrace, "loader")
		return wantErr
	})

	_, err := sectionload.All(
		ctx,
		exec,
		loaderHookNewLoadableParentQuery(loader),
		loaderHookFakeParentParts,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("sectionload.All: got err %v, want wrapping %v", err, wantErr)
	}
	if len(loaderHookTrace) != 1 || loaderHookTrace[0] != "loader" {
		t.Fatalf("unexpected trace on loader error (the hook may have run): got %v want [loader]", loaderHookTrace)
	}
}

// TestParentHookErrorAfterLoaderRan asserts that a hook error occurs only
// after the loader has already run: the loader is never skipped merely
// because the hook is destined to fail.
func TestParentHookErrorAfterLoaderRan(t *testing.T) {
	loaderHookTrace = nil
	wantErr := errors.New("hook boom")
	errLoaderHook = wantErr
	loaderHookRegisterAfterSelectHook()
	defer func() { errLoaderHook = nil }()
	ctx := context.Background()
	exec := &loaderHookFakeExecutor{}

	loader := bob.LoaderFunc(func(_ context.Context, _ bob.Executor, _ any) error {
		loaderHookTrace = append(loaderHookTrace, "loader")
		return nil
	})

	_, err := sectionload.All(
		ctx,
		exec,
		loaderHookNewLoadableParentQuery(loader),
		loaderHookFakeParentParts,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("sectionload.All: got err %v, want wrapping %v", err, wantErr)
	}
	if len(loaderHookTrace) != 2 || loaderHookTrace[0] != "loader" || loaderHookTrace[1] != "hook" {
		t.Fatalf("the loader should run first even when the hook errors: got %v want [loader hook]", loaderHookTrace)
	}
}
