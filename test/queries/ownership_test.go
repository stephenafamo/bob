package queries_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql"
	my "github.com/stephenafamo/bob/dialect/mysql/dialect"
	mi "github.com/stephenafamo/bob/dialect/mysql/im"
	ms "github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/psql"
	pg "github.com/stephenafamo/bob/dialect/psql/dialect"
	pi "github.com/stephenafamo/bob/dialect/psql/im"
	pm "github.com/stephenafamo/bob/dialect/psql/mm"
	ps "github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/sqlite"
	sq "github.com/stephenafamo/bob/dialect/sqlite/dialect"
	si "github.com/stephenafamo/bob/dialect/sqlite/im"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

// Edit inherited slots, then reinstall the same mod and edit its new slots.
// Both the parent and a new query built from the mod must remain usable.
func checkReuse[Q any](t *testing.T, build func(...bob.Mod[Q]) bob.QueryBuilder[Q], mod bob.Mod[Q], edit bob.ModFunc[Q]) {
	t.Helper()
	ctx := t.Context()
	parent := build(mod)
	wantSQL, wantArgs := parent.MustBuild(ctx, nil)
	for _, ms := range [][]bob.Mod[Q]{{edit}, {mod, edit}} {
		child := parent.WithMods(ms...)
		if sql, args := child.MustBuild(ctx, nil); sql == wantSQL && slices.Equal(args, wantArgs) {
			t.Fatal("edit did not change the child")
		}
		for _, q := range []bob.QueryBuilder[Q]{parent, build(mod)} {
			if sql, args := q.MustBuild(ctx, nil); sql != wantSQL || !slices.Equal(args, wantArgs) {
				t.Fatalf("reused mod changed another query: got %q %v; want %q %v", sql, args, wantSQL, wantArgs)
			}
		}
	}
}

func checkInsertMods[Q interface {
	RowCount() int
	SetValue(int, int, bob.Expression)
	SetTableColumns(...string)
}](t *testing.T, build func(...bob.Mod[Q]) bob.QueryBuilder[Q], into bob.Mod[Q],
	values func(...bob.Expression) bob.Mod[Q], rows func(...[]bob.Expression) bob.Mod[Q],
) {
	t.Helper()
	for _, mod := range []bob.Mod[Q]{values(expr.Arg(1)), rows([]bob.Expression{expr.Arg(1)}, []bob.Expression{expr.Arg(2)})} {
		checkReuse(t, build, bob.Mods[Q]{into, mod}, func(q Q) {
			q.SetValue(q.RowCount()-1, 0, expr.Arg(99))
		})
	}
	checkReuse(t, build, into, func(q Q) { q.SetTableColumns("different_column") })
}

func TestInsertModOwnership(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		checkInsertMods(t, mysql.Insert, mi.Into("jobs", "id"), mi.Values, mi.Rows)
	})
	t.Run("postgres", func(t *testing.T) {
		for _, into := range []bob.Mod[pg.InsertEditor]{pi.Into("jobs", "id"), pi.IntoAs("jobs", "j", "id")} {
			checkInsertMods(t, psql.Insert, into, pi.Values, pi.Rows)
		}
	})
	t.Run("sqlite", func(t *testing.T) {
		for _, into := range []bob.Mod[sq.InsertEditor]{si.Into("jobs", "id"), si.IntoAs("jobs", "j", "id")} {
			checkInsertMods(t, sqlite.Insert, into, si.Values, si.Rows)
		}
	})
}

func TestStandaloneRowOwnership(t *testing.T) {
	checkReuse(t, mysql.Values, mods.Values[my.ValuesEditor]{expr.Arg(1)}, func(q my.ValuesEditor) { q.SetValue(q.RowCount()-1, 0, expr.Arg(2)) })
	checkReuse(t, psql.Values, mods.Values[pg.ValuesEditor]{expr.Arg(1)}, func(q pg.ValuesEditor) { q.SetValue(q.RowCount()-1, 0, expr.Arg(2)) })
	checkReuse(t, sqlite.Values, mods.Values[sq.ValuesEditor]{expr.Arg(1)}, func(q sq.ValuesEditor) { q.SetValue(q.RowCount()-1, 0, expr.Arg(2)) })
}

func TestNestedClauseOwnership(t *testing.T) {
	// Each query must retain its snapshot after its input clauses change.
	check := func(q bob.Query, edit func()) {
		t.Helper()
		sql, args := bob.MustBuild(t.Context(), nil, q)
		edit()
		if got, vals := bob.MustBuild(t.Context(), nil, q); got != sql || !slices.Equal(args, vals) {
			t.Fatalf("changed from %s %v to %s %v", sql, args, got, vals)
		}
	}
	group := ps.Grouping("city")
	set := ps.GroupingSets(ps.Rollup("country", &group))
	lock := &clause.Lock{Strength: clause.LockStrengthUpdate, Tables: []any{"jobs"}}
	join := clause.Join{Type: clause.InnerJoin, To: clause.TableRef{Expression: "owners"}, Using: []string{"id"}}
	materialized := true
	cte := clause.CTE{Name: "jobs", Columns: []string{"id"}, Query: psql.Select(ps.Columns("1")), Materialized: &materialized}
	win := clause.NamedWindow{Name: "w"}
	win.Definition.AppendOrder(expr.Raw("id"))
	query := psql.Select(ps.GroupBy(&set), bob.ModFunc[pg.SelectEditor](func(e pg.SelectEditor) {
		e.AppendLock(lock)
		e.SetTable("jobs")
		e.AppendJoin(join)
		e.AppendCTE(cte)
		e.AppendWindow(win)
	}))
	check(query, func() {
		group.Groups[0] = "region"
		lock.Tables[0] = "owners"
		join.Using[0] = "owner_id"
		cte.Columns[0] = "job_id"
		materialized = false
		win.Definition.Expressions[0] = expr.Raw("job_id")
	})
	hint := clause.IndexHint{Type: "USE", Indexes: []string{"id"}}
	check(
		mysql.Select(ms.From("jobs"), bob.ModFunc[my.SelectEditor](func(e my.SelectEditor) { e.AppendIndexHint(hint) })),
		func() { hint.Indexes[0] = "by_owner" },
	)
	conflict := clause.ConflictClause{Do: "NOTHING", Target: clause.ConflictTarget{Columns: []any{"id"}}}
	check(
		psql.Insert(pi.Into("jobs"), bob.ModFunc[pg.InsertEditor](func(e pg.InsertEditor) { e.SetConflict(conflict) })),
		func() { conflict.Target.Columns[0] = "job_id" },
	)
	when := pg.MergeWhen{
		Type:   pg.MergeWhenNotMatched,
		Action: pg.MergeAction{Type: pg.MergeActionInsert, Columns: []string{"id"}, Values: []bob.Expression{expr.Arg(1)}},
	}
	check(
		psql.Merge(pm.Into("jobs"), pm.Using("src").On(expr.Raw("true")), bob.ModFunc[pg.MergeEditor](func(e pg.MergeEditor) { e.AppendWhen(when) })),
		func() { when.Action.Columns[0] = "job_id"; when.Action.Values[0] = expr.Arg(2) },
	)
}

func TestPostgresAppendOnlyBranches(t *testing.T) {
	// Three successive appends leave room for a fourth entry in both lists.
	base := psql.Insert(pi.Into("jobs"),
		pi.Values(expr.Arg(0)), pi.Values(expr.Arg(0)), pi.Values(expr.Arg(1)),
		pi.Returning("x"), pi.Returning("y"), pi.Returning("id"))
	left := base.WithMods(pi.Values(expr.Arg(2)), pi.Returning("a"))
	right := base.WithMods(pi.Rows([]bob.Expression{expr.Arg(3)}), pi.Returning("b"))
	for _, tc := range []struct {
		q         psql.InsertQuery
		returning string
		args      []any
	}{
		{base, "x, y, id", []any{0, 0, 1}},
		{left, "x, y, id, a", []any{0, 0, 1, 2}},
		{right, "x, y, id, b", []any{0, 0, 1, 3}},
	} {
		sql, args := tc.q.MustBuild(t.Context(), nil)
		if !strings.HasSuffix(strings.TrimSpace(sql), "RETURNING "+tc.returning) || !slices.Equal(args, tc.args) {
			t.Fatalf("got %q %v; want RETURNING %s and args %v", sql, args, tc.returning, tc.args)
		}
	}
}
