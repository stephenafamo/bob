package queries_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	my "github.com/stephenafamo/bob/dialect/mysql/dialect"
	mf "github.com/stephenafamo/bob/dialect/mysql/fm"
	"github.com/stephenafamo/bob/dialect/psql"
	pf "github.com/stephenafamo/bob/dialect/psql/fm"
	"github.com/stephenafamo/bob/dialect/sqlite"
	sq "github.com/stephenafamo/bob/dialect/sqlite/dialect"
	sf "github.com/stephenafamo/bob/dialect/sqlite/fm"
	"github.com/stephenafamo/bob/expr"
)

func TestFunctionBranches(t *testing.T) {
	myCount, sqCount, pgCount := mysql.F("count", "id"), sqlite.F("count", "id"), psql.F("count", "id")
	for _, tc := range []struct{ base, distinct bob.Query }{
		{mysql.RawQuery("SELECT ?", myCount), mysql.RawQuery("SELECT ?", myCount(mf.Distinct()))},
		{sqlite.RawQuery("SELECT ?", sqCount), sqlite.RawQuery("SELECT ?", sqCount(sf.Distinct()))},
		{psql.RawQuery("SELECT ?", pgCount), psql.RawQuery("SELECT ?", pgCount(pf.Distinct()))},
	} {
		base, _ := bob.MustBuild(t.Context(), nil, tc.base)
		child, _ := bob.MustBuild(t.Context(), nil, tc.distinct)
		if strings.Contains(base, "DISTINCT") || !strings.Contains(child, "DISTINCT") {
			t.Fatalf("base: %q; derived: %q", base, child)
		}
	}
}

func contextualInsert[Q interface {
	AppendContextualModFunc(func(context.Context, Q) (context.Context, error))
}](t *testing.T, base bob.QueryBuilder[Q], set func(Q, int)) {
	t.Helper()
	type tenantKey struct{}
	byTenant := base.WithMods(bob.ModFunc[Q](func(q Q) {
		q.AppendContextualModFunc(func(ctx context.Context, ast Q) (context.Context, error) {
			set(ast, ctx.Value(tenantKey{}).(int))
			return ctx, nil
		})
	}))
	for _, tenant := range []int{7, 11} {
		t.Run("rebuild", func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(t.Context(), tenantKey{}, tenant)
			for range 10 {
				_, args := byTenant.MustBuild(ctx, nil)
				if !slices.Equal(args, []any{tenant}) {
					t.Fatalf("got %v; want %d", args, tenant)
				}
				_, args = byTenant.MustBuild(context.WithValue(ctx, bob.SkipContextualModsKey{}, true), nil)
				if !slices.Equal(args, []any{1}) {
					t.Fatalf("render changed stored row: %v", args)
				}
			}
		})
	}
}

func TestContextualRows(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		q := mysql.Insert(bob.ModFunc[my.InsertEditor](func(q my.InsertEditor) { q.SetTable("jobs"); q.AppendValues(mysql.Arg(1)) }))
		contextualInsert(t, q, func(q my.InsertEditor, n int) { q.SetValue(0, 0, mysql.Arg(n)) })
	})
	t.Run("sqlite", func(t *testing.T) {
		q := sqlite.Insert(bob.ModFunc[sq.InsertEditor](func(q sq.InsertEditor) { q.SetTable("jobs"); q.AppendValues(sqlite.Arg(1)) }))
		contextualInsert(t, q, func(q sq.InsertEditor, n int) { q.SetValue(0, 0, sqlite.Arg(n)) })
	})
}

func TestRawAndCachedArguments(t *testing.T) {
	for _, raw := range []func(string, ...any) bob.ExpressionQuery[expr.Clause]{sqlite.RawQuery, mysql.RawQuery} {
		input := []any{1}
		q := raw("SELECT ?", input...)
		input[0] = 2
		cached, err := q.CacheN(t.Context(), nil, 3)
		if err != nil {
			t.Fatal(err)
		}
		_, args := cached.MustBuildN(t.Context(), nil, 3)
		if !slices.Equal(args, []any{1}) {
			t.Fatal(args)
		}
		args[0] = 3
		_, args = cached.MustBuildN(t.Context(), nil, 3)
		if !slices.Equal(args, []any{1}) {
			t.Fatalf("cached arguments changed: %v", args)
		}
	}
}

// Replace an existing cell (not just a slice header) while other callers
// derive from and render the same parent.
func branches[Q any](t *testing.T, newQuery func(...bob.Mod[Q]) bob.QueryBuilder[Q], seed func(Q), replace func(Q, int)) {
	t.Helper()
	base := newQuery(bob.ModFunc[Q](seed))
	build := func(t *testing.T, q bob.QueryBuilder[Q], want int) {
		t.Helper()
		sql, args := q.MustBuildN(t.Context(), nil, 3)
		if sql == "" || !slices.Equal(args, []any{want}) {
			t.Fatalf("got %q %v; want argument %d", sql, args, want)
		}
	}
	for _, n := range []int{17, 41} {
		t.Run("sibling", func(t *testing.T) {
			t.Parallel()
			for range 10 {
				child := base.WithMods(bob.ModFunc[Q](func(q Q) { replace(q, n) }))
				build(t, child, n)
				build(t, base, 1)
			}
		})
	}
}

func TestMySQLBranches(t *testing.T) {
	t.Run("select", func(t *testing.T) {
		branches(t, mysql.Select, func(q my.SelectEditor) { q.AppendSelect(mysql.Arg(1)) },
			func(q my.SelectEditor, n int) { q.SetSelect(mysql.Arg(n)) })
	})
	t.Run("insert", func(t *testing.T) {
		branches(t, mysql.Insert, func(q my.InsertEditor) { q.SetTable("jobs"); q.AppendValues(mysql.Arg(1)) },
			func(q my.InsertEditor, n int) { q.SetValue(0, 0, mysql.Arg(n)) })
	})
	t.Run("update", func(t *testing.T) {
		branches(t, mysql.Update, func(q my.UpdateEditor) { q.SetTable("jobs"); q.AppendSet("active = 1"); q.AppendWhere(mysql.Arg(1)) },
			func(q my.UpdateEditor, n int) { q.SetWhereAt(0, mysql.Arg(n)) })
	})
	t.Run("delete", func(t *testing.T) {
		branches(t, mysql.Delete, func(q my.DeleteEditor) { q.AppendTable(mysql.Quote("jobs")); q.AppendWhere(mysql.Arg(1)) },
			func(q my.DeleteEditor, n int) { q.SetWhereAt(0, mysql.Arg(n)) })
	})
	t.Run("values", func(t *testing.T) {
		branches(t, mysql.Values, func(q my.ValuesEditor) { q.AppendValues(mysql.Arg(1)) },
			func(q my.ValuesEditor, n int) { q.SetValue(0, 0, mysql.Arg(n)) })
	})
}

func TestSQLiteBranches(t *testing.T) {
	t.Run("select", func(t *testing.T) {
		branches(t, sqlite.Select, func(q sq.SelectEditor) { q.AppendSelect(sqlite.Arg(1)) },
			func(q sq.SelectEditor, n int) { q.SetSelect(sqlite.Arg(n)) })
	})
	t.Run("insert", func(t *testing.T) {
		branches(t, sqlite.Insert, func(q sq.InsertEditor) { q.SetTable("jobs"); q.AppendValues(sqlite.Arg(1)) },
			func(q sq.InsertEditor, n int) { q.SetValue(0, 0, sqlite.Arg(n)) })
	})
	t.Run("update", func(t *testing.T) {
		branches(t, sqlite.Update, func(q sq.UpdateEditor) {
			q.SetTargetName("jobs")
			q.AppendSet("active = 1")
			q.AppendWhere(sqlite.Arg(1))
		},
			func(q sq.UpdateEditor, n int) { q.SetWhereAt(0, sqlite.Arg(n)) })
	})
	t.Run("delete", func(t *testing.T) {
		branches(t, sqlite.Delete, func(q sq.DeleteEditor) { q.SetTable("jobs"); q.AppendWhere(sqlite.Arg(1)) },
			func(q sq.DeleteEditor, n int) { q.SetWhereAt(0, sqlite.Arg(n)) })
	})
	t.Run("values", func(t *testing.T) {
		branches(t, sqlite.Values, func(q sq.ValuesEditor) { q.AppendValues(sqlite.Arg(1)) },
			func(q sq.ValuesEditor, n int) { q.SetValue(0, 0, sqlite.Arg(n)) })
	})
}
