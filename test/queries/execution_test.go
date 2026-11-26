package queries_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
	_ "modernc.org/sqlite"
)

// Both overrides must run even though the embedded query has model hooks.
type wrappedSelect struct{ sqlite.SelectQuery }

type incrementKey struct{}

// Embeds only the Query interface: its concrete implementation is not a builder.
type replacingQuery struct {
	bob.Query
	run func(context.Context, bob.Executor) (context.Context, bob.Query, error)
}

func (q replacingQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, bob.Query, error) {
	return q.run(ctx, exec)
}

func TestCustomHookableQuery(t *testing.T) {
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := t.Context()
	runs, loads := 0, 0
	replacement := sqlite.Select(sm.Columns(bob.Named("value")), bob.ModFunc[dialect.SelectEditor](func(ast dialect.SelectEditor) {
		ast.AppendHooks(func(ctx context.Context, _ bob.Executor) (context.Context, error) {
			t.Fatal("ran hooks on a replacement statement")
			return ctx, nil
		})
		ast.AppendLoader(bob.LoaderFunc(func(context.Context, bob.Executor, any) error { loads++; return nil }))
	}))
	var q bob.Query = replacingQuery{
		Query: sqlite.Select(sm.Columns("-1")),
		run: func(ctx context.Context, _ bob.Executor) (context.Context, bob.Query, error) {
			runs++
			return context.WithValue(ctx, incrementKey{}, runs), replacement, nil
		},
	}
	check := func(n int, err error) {
		t.Helper()
		if err != nil || n != 42 {
			t.Fatalf("got %d, %v; want 42", n, err)
		}
	}
	hctx, stmt, err := bob.RunQueryHooks(ctx, db, q)
	if err != nil || hctx.Value(incrementKey{}) != 1 {
		t.Fatalf("expected the hook's context, got %v, %v", hctx, err)
	}
	mapper := scan.SingleColumnMapper[int]
	check(bob.One(hctx, db, bob.BindNamed(hctx, stmt, 42), mapper))
	bound := bob.BindNamed(ctx, q, 42)
	check(bob.One(ctx, db, bound, mapper))
	outer := bob.QueryFromExpression(q, dialect.Dialect, bob.QueryTypeSelect)
	check(bob.One(ctx, db, bob.BindNamed(ctx, outer, 42), mapper))
	cached, err := bob.Cache(ctx, db, bound)
	if err != nil {
		t.Fatal(err)
	}
	check(bob.One(ctx, db, cached, mapper))
	prepared, err := bob.PrepareQuery[int](ctx, db, q, mapper)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Close()
	check(prepared.One(ctx, 42))
	if runs != 5 || loads != 5 {
		t.Fatalf("hook ran %d times and loader ran %d times; want 5 each", runs, loads)
	}
}

func TestCustomQueryHookError(t *testing.T) {
	want := errors.New("hook rejected query")
	q := replacingQuery{run: func(ctx context.Context, _ bob.Executor) (context.Context, bob.Query, error) {
		return ctx, nil, want
	}}
	if _, err := bob.Exec(t.Context(), nil, q); !errors.Is(err, want) {
		t.Fatalf("got %v; want hook error", err)
	}
}

type contextOnlyQuery struct{ bob.Query }

func (contextOnlyQuery) RunHooks(context.Context, bob.Executor) (context.Context, error) {
	panic("context-only RunHooks must not be dispatched as a query hook")
}

func TestContextOnlyMethodDoesNotMakeQueryHookable(t *testing.T) {
	q := &contextOnlyQuery{Query: sqlite.Select(sm.Columns("1"))}
	ctx := t.Context()
	gotCtx, stmt, err := bob.RunQueryHooks(ctx, nil, q)
	if err != nil || gotCtx != ctx || stmt != q {
		t.Fatalf("expected unchanged query and context; got %T, %v", stmt, err)
	}
}

func TestRunQueryHooksReturnsExecutableBranch(t *testing.T) {
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	embedded, registered := 0, 0
	q := dialect.NewSelect([]bob.QueryHook[sqlite.SelectEditor]{
		func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
			registered++
			sm.Where(sqlite.Arg(registered)).Edit(q)
			return ctx, nil
		},
	}, sm.Columns("1"), bob.ModFunc[dialect.SelectEditor](func(e dialect.SelectEditor) {
		e.AppendHooks(func(ctx context.Context, _ bob.Executor) (context.Context, error) {
			embedded++
			return context.WithValue(ctx, incrementKey{}, embedded), nil
		})
	}))
	for run := 1; run <= 2; run++ {
		ctx, stmt, err := q.RunHooks(t.Context(), db)
		if err != nil {
			t.Fatal(err)
		}
		if _, args := bob.MustBuild(ctx, nil, stmt); len(args) != 1 || args[0] != run {
			t.Fatalf("run %d: got args %v", run, args)
		}
		if embedded != run-1 {
			t.Fatal("standalone build ran a before-model callback")
		}
		if n, err := bob.One(ctx, db, stmt.(sqlite.SelectQuery).WithMods(sm.Limit(1)), scan.SingleColumnMapper[int]); err != nil || n != 1 {
			t.Fatalf("derived statement: got %d, %v", n, err)
		}
		if embedded != run || registered != run {
			t.Fatalf("run %d: embedded=%d, registered=%d", run, embedded, registered)
		}
	}
}

func (w wrappedSelect) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, bob.Query, error) {
	ctx, stmt, err := w.SelectQuery.RunHooks(context.WithValue(ctx, incrementKey{}, 1), exec)
	if err != nil {
		return ctx, nil, err
	}
	w.SelectQuery = stmt.(sqlite.SelectQuery)
	return ctx, w, nil
}

func (w wrappedSelect) WriteQuery(ctx context.Context, out io.StringWriter, start int) ([]any, error) {
	out.WriteString("SELECT 1 + (")
	args, err := w.SelectQuery.WriteQuery(ctx, out, start)
	out.WriteString(")")
	return args, err
}

func TestExecuteWrappedQuery(t *testing.T) {
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var runs, loads, maps int
	hook := func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
		runs++
		if ctx.Value(incrementKey{}) != 1 {
			t.Fatal("model hook did not receive the wrapper hook's context")
		}
		q.SetSelect(bob.Named("value"))
		q.AppendLoader(bob.LoaderFunc(func(context.Context, bob.Executor, any) error { loads++; return nil }))
		q.AppendMapperMod(func(context.Context, []string) (scan.BeforeFunc, scan.AfterMod) {
			maps++
			return func(*scan.Row) (any, error) { return nil, nil }, func(any, any) error { return nil }
		})
		return ctx, nil
	}
	ctx := t.Context()
	q := wrappedSelect{dialect.NewSelect([]bob.QueryHook[sqlite.SelectEditor]{hook}, sm.Columns("0"))}
	bound := bob.BindNamed(ctx, q, 41)
	mapper := scan.SingleColumnMapper[int]
	check := func(n int, err error) {
		t.Helper()
		if err != nil || n != 42 {
			t.Fatalf("got %d, %v; want 42", n, err)
		}
	}
	check(bob.One(ctx, db, bound, mapper))

	cached, err := bob.Cache(ctx, db, bound)
	if err != nil {
		t.Fatal(err)
	}
	check(bob.One(ctx, db, cached, mapper))

	stmt, err := bob.PrepareQuery[int](ctx, db, q, mapper)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	check(stmt.One(ctx, 41))
	check(stmt.One(ctx, 41))
	if runs != 3 || loads != 4 || maps != 4 {
		t.Fatalf("hook runs=%d loads=%d mapper mods=%d; want 3, 4, 4", runs, loads, maps)
	}
	if len(q.GetLoaders(ctx)) != 0 || len(q.GetMapperMods(ctx)) != 0 {
		t.Fatal("hook callbacks leaked into the reusable query")
	}
}

// Older wrappers can override either callback getter without taking a context.
type legacyCallbacks struct {
	bob.Query
	loader bob.Loader
	mod    scan.MapperMod
}

func (q legacyCallbacks) GetLoaders() []bob.Loader        { return []bob.Loader{q.loader} }
func (q legacyCallbacks) GetMapperMods() []scan.MapperMod { return []scan.MapperMod{q.mod} }

func TestCachePreservesLegacyWrapperCallbacks(t *testing.T) {
	q := legacyCallbacks{Query: sqlite.Select(sm.Columns("1"))}
	cached, err := bob.Cache(t.Context(), nil, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(cached.GetLoaders(t.Context())) != 1 || len(cached.GetMapperMods(t.Context())) != 1 {
		t.Fatal("cache lost the wrapper's callback overrides")
	}
}

func TestCustomAfterPreloaderIsolatesExecutions(t *testing.T) {
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	p := orm.NewAfterPreloader[int, []int]()
	var batches []any
	p.AppendLoader(bob.LoaderFunc(func(_ context.Context, _ bob.Executor, batch any) error {
		batches = append(batches, batch)
		return nil
	}))

	var inner sqlite.SelectQuery
	base := sqlite.Select(bob.ModFunc[sqlite.SelectEditor](func(e sqlite.SelectEditor) {
		e.AppendLoader(p)
		e.AppendMapperMod(func(ctx context.Context, _ []string) (scan.BeforeFunc, scan.AfterMod) {
			return func(*scan.Row) (any, error) { return nil, nil }, func(_, row any) error {
				if err := p.Collect(ctx, row); err != nil {
					return err
				}
				if row == 1 {
					// Run One while All still has a row waiting for its loader.
					_, err := bob.One(ctx, db, inner, scan.SingleColumnMapper[int])
					return err
				}
				return nil
			}
		})
	}))
	inner = base.WithMods(sm.Columns("3"))
	outer := base.WithMods(sm.Columns("column1"), sm.From("(VALUES (1), (2))"), sm.OrderBy("column1"))
	if _, err := bob.All(t.Context(), db, outer, scan.SingleColumnMapper[int]); err != nil {
		t.Fatal(err)
	}
	if want := []any{3, []int{1, 2}}; !reflect.DeepEqual(batches, want) {
		t.Fatalf("loaded batches: got %v, want %v", batches, want)
	}
}
