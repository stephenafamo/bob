package queries_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/scan"
)

func TestQueryHooksComposeAtExecution(t *testing.T) {
	type row struct{ ID int }
	view := sqlite.NewView[row]("", "jobs", expr.ColsForStruct[row]("jobs"))
	type tenantKey struct{}
	type seenKey struct{}
	view.SelectQueryHooks.AppendHooks(
		func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
			tenant := ctx.Value(tenantKey{}).(int)
			sm.Columns(sqlite.Arg(tenant)).Edit(q)
			return context.WithValue(ctx, seenKey{}, tenant+1), nil
		},
		func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
			sm.Columns(sqlite.Arg(ctx.Value(seenKey{}))).Edit(q)
			return ctx, nil
		},
	)
	base := view.Query(sm.Columns(sqlite.Arg(1)))
	view.SelectQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
		sm.Columns(sqlite.Arg(99)).Edit(q)
		return ctx, nil
	})
	// Even a branch derived after registration keeps its parent's hook list.
	branch := base.WithMods(sm.Columns(sqlite.Arg(2)))
	fresh := view.Query(sm.Columns(sqlite.Arg(1)))
	for _, tenant := range []int{10, 20} {
		t.Run("execution", func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(t.Context(), tenantKey{}, tenant)
			check := func(ctx context.Context, q bob.Query, want ...any) {
				t.Helper()
				if _, args := bob.MustBuild(ctx, nil, q); !slices.Equal(args, want) {
					t.Fatalf("got arguments %v; want %v", args, want)
				}
			}
			// Even if a previous execution's context is reused, start at the original branch.
			for range 2 {
				next, stmt, err := branch.RunHooks(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				ctx = next
				check(ctx, stmt, 1, 2, tenant, tenant+1)
				check(ctx, base, 1, tenant, tenant+1)
				check(ctx, branch, 1, 2, tenant, tenant+1)
			}
			ctx, stmt, err := branch.RunHooks(bob.SkipQueryHooks(ctx), nil)
			if err != nil {
				t.Fatal(err)
			}
			check(ctx, stmt, 1, 2)
			ctx, stmt, err = fresh.RunHooks(context.WithValue(t.Context(), tenantKey{}, tenant), nil)
			if err != nil {
				t.Fatal(err)
			}
			check(ctx, stmt, 1, tenant, tenant+1, 99)
		})
	}
}

func TestQueryHookErrorStopsExecution(t *testing.T) {
	want := errors.New("hook failed")
	q := dialect.NewSelect([]bob.QueryHook[sqlite.SelectEditor]{
		func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
			sm.Limit(0).Edit(q)
			return ctx, want
		},
		func(ctx context.Context, _ bob.Executor, q sqlite.SelectEditor) (context.Context, error) {
			t.Fatal("ran a hook after an error")
			return ctx, nil
		},
	})
	if _, err := q.Exec(t.Context(), nil); !errors.Is(err, want) {
		t.Fatalf("got %v; want hook error", err)
	}
}

func TestBuildHooksUseTransactionInSubqueries(t *testing.T) {
	ctx := t.Context()
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.ExecContext(ctx, "CREATE TEMP TABLE tenant (id INTEGER); INSERT INTO tenant VALUES (42)"); err != nil {
		t.Fatal(err)
	}

	needTx := errors.New("tenant lookup needs a transaction")
	var calls int
	inner := dialect.NewSelect([]bob.QueryHook[sqlite.SelectEditor]{
		func(ctx context.Context, exec bob.Executor, e sqlite.SelectEditor) (context.Context, error) {
			calls++
			if exec == nil {
				return ctx, needTx
			}
			if exec != tx {
				t.Fatal("hook received a different executor")
			}
			id, err := bob.One(ctx, exec, sqlite.RawQuery("SELECT id FROM tenant"), scan.SingleColumnMapper[int])
			e.SetSelect(sqlite.Arg(id))
			return ctx, err
		},
	})
	for _, q := range []sqlite.SelectQuery{
		inner,
		sqlite.Select(sm.Columns(inner.As("id"))),
		sqlite.Select(sm.With("scope", "id").As(inner), sm.Columns("id"), sm.From("scope")),
	} {
		calls = 0
		if _, _, err := q.Build(ctx, nil); !errors.Is(err, needTx) {
			t.Fatalf("offline build: got %v, want %v", err, needTx)
		}
		sql, args, err := bob.BuildN(ctx, tx, q, 3)
		if err != nil || !strings.Contains(sql, "?3") || !slices.Equal(args, []any{42}) {
			t.Fatalf("transaction build: %q, %v, %v", sql, args, err)
		}
		if id, err := bob.One(ctx, tx, q, scan.SingleColumnMapper[int]); err != nil || id != 42 {
			t.Fatalf("execution: got %d, %v", id, err)
		}
		if calls != 3 {
			t.Fatalf("got %d hook calls for two builds and one execution", calls)
		}
	}
}
