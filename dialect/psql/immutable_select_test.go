package psql

import (
	"context"
	"slices"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type selectRecorder struct {
	bob.Executor
	query string
	args  []any
}

func (r *selectRecorder) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	r.query, r.args = query, args
	return r.Executor.QueryContext(ctx, query, args...)
}

func TestSelectExecutionIsolation(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE jobs (tenant_id int, owner_id int, active bool);
		INSERT INTO jobs VALUES (1, 11, true), (1, 11, false), (2, 11, true), (1, 22, true)`); err != nil {
		t.Fatal(err)
	}
	view := NewViewx[int, []int]("", "jobs", expr.NewColumnsExpr("owner_id"), scan.SingleColumnMapper[int])
	hooks, loads := 0, 0
	view.SelectQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q SelectEditor) (context.Context, error) {
		hooks++
		// Write an existing slot as well as append. A mere struct copy leaks this edit.

		q.SetWhereAt(0, Quote("tenant_id").EQ(Arg(2)))
		q.AppendLoader(bob.LoaderFunc(func(context.Context, bob.Executor, any) error {
			loads++
			return nil
		}))
		sm.Where(Quote("active").EQ(Arg(true))).Edit(q)
		return ctx, nil
	})
	base := view.Query(sm.Where(Quote("tenant_id").EQ(Arg(1))))
	left := base.WithMods(sm.Where(Quote("owner_id").EQ(Arg(11))), sm.OrderBy("owner_id"), sm.Limit(0))
	right := base.WithMods(sm.Where(Quote("owner_id").EQ(Arg(22))))
	check := func() {
		t.Helper()
		for _, tc := range []struct {
			q    bob.Query
			args []any
		}{{base, []any{1}}, {left, []any{1, 11}}, {right, []any{1, 22}}} {
			want := "SELECT \n\"owner_id\"\nFROM \"jobs\"\nWHERE (\"tenant_id\" = $1)"
			if tc.q != base {
				want += ` AND ("owner_id" = $2)`
			}
			if tc.q == left {
				want += "\nORDER BY owner_id\nLIMIT 0"
			}
			want += "\n"
			sql, args := bob.MustBuild(bob.SkipQueryHooks(ctx), tx, tc.q)
			if sql != want || !slices.Equal(args, tc.args) {
				t.Fatalf("query changed: %s %v; want %s %v", sql, args, want, tc.args)
			}
		}
	}
	check()
	exec := &selectRecorder{Executor: tx}
	for range 2 {
		n, err := left.Count(ctx, exec)
		if err != nil || n != 1 {
			t.Fatalf("left.Count: %d, %v", n, err)
		}
		want := "SELECT \ncount(1)\nFROM \"jobs\"\nWHERE (\"tenant_id\" = $1) AND (\"owner_id\" = $2) AND (\"active\" = $3)\n"
		if exec.query != want || !slices.Equal(exec.args, []any{2, 11, true}) {
			t.Fatalf("Count sent %q %v; want %q [2 11 true]", exec.query, exec.args, want)
		}
		check()
		rows, err := right.All(ctx, tx)
		if err != nil || len(rows) != 0 {
			t.Fatalf("right.All: %v, %v", rows, err)
		}
		check()
	}
	if hooks != 4 || loads != 2 {
		t.Fatalf("got %d hooks and %d loads; want 4 hooks and 2 model loads", hooks, loads)
	}
	cached, err := right.CacheN(ctx, tx, 3)
	if err != nil {
		t.Fatal(err)
	}
	query, args := cached.MustBuildN(ctx, nil, 3)
	want := "SELECT \n\"owner_id\"\nFROM \"jobs\"\nWHERE (\"tenant_id\" = $3) AND (\"owner_id\" = $4) AND (\"active\" = $5)\n"
	if query != want || !slices.Equal(args, []any{2, 22, true}) {
		t.Fatalf("CacheN built %q %v; want %q [2 22 true]", query, args, want)
	}
	hookCtx, stmt, err := right.RunHooks(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	if query, args = bob.MustBuildN(hookCtx, nil, stmt, 3); query != want || !slices.Equal(args, []any{2, 22, true}) {
		t.Fatalf("RunHooks/BuildN built %q %v; want %q [2 22 true]", query, args, want)
	}
	check()
}

type branchEmployee struct {
	ID int `db:"id"`
}

func (*branchEmployee) Preload(string, any) error { return nil }

// Start a second branch after the first one has scanned its rows but before
// its nested loader receives them. This makes a shared row buffer observable
// without relying on goroutine timing.
func TestSelectPreloadBranchIsolation(t *testing.T) {
	ctx := t.Context()
	employees := NewView[*branchEmployee]("", "employees", expr.ColsForStruct[branchEmployee]("employees"))
	input := sm.With("employees", "id", "manager_id").As(Select(sm.Columns("*"), sm.From("(VALUES (1, 11), (2, 22), (11, 11), (22, 22)) AS e")))
	var loaded []int
	managers := Preload[*branchEmployee, []*branchEmployee](PreloadRel{Name: "Manager", Sides: []PreloadSide{{
		From: employees, To: employees, FromColumns: []string{"manager_id"}, ToColumns: []string{"id"},
	}}}, []string{"id"}, nil, orm.Loader[dialect.SelectEditor](func(_ context.Context, _ bob.Executor, got any) error {
		manager, ok := got.(*branchEmployee)
		if !ok {
			t.Fatalf("nested loader received %T %v; want one manager", got, got)
		}
		loaded = append(loaded, manager.ID)
		return nil
	}))
	var sibling *ViewQuery[*branchEmployee, []*branchEmployee]
	base := employees.Query(input, orm.Loader[dialect.SelectEditor](func(ctx context.Context, exec bob.Executor, got any) error {
		if got.(*branchEmployee).ID == 1 {
			_, err := sibling.One(ctx, exec)
			return err
		}
		return nil
	}), managers)
	sibling = base.WithMods(sm.Where(Quote("employees", "id").EQ(Arg(2))))
	left := base.WithMods(sm.Where(Quote("employees", "id").EQ(Arg(1))))
	if _, err := left.One(ctx, testDB); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(loaded, []int{22, 11}) {
		t.Fatalf("nested loaders received manager IDs %v; want [22 11]", loaded)
	}
}

func TestSelectContextualAndCustomMods(t *testing.T) {
	type ownerKey struct{}
	base := Select(sm.From("jobs"), sm.Where(Quote("tenant_id").EQ(Arg(1))))
	left := base.WithMods(bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetWhereAt(0, Quote("tenant_id").EQ(Arg(2)))
		q.AppendContextualModFunc(func(ctx context.Context, q dialect.SelectEditor) (context.Context, error) {
			q.SetWhereAt(0, Quote("owner_id").EQ(Arg(ctx.Value(ownerKey{}))))
			return ctx, nil
		})
	}))
	for _, owner := range []int{11, 22, 33, 44} {
		t.Run("build", func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(t.Context(), ownerKey{}, owner)
			for range 20 {
				want, _ := Select(sm.From("jobs"), sm.Where(Quote("owner_id").EQ(Arg(owner)))).MustBuild(ctx, nil)
				got, args := left.MustBuild(ctx, nil)
				if got != want || !slices.Equal(args, []any{owner}) {
					t.Fatalf("build: %s %v", got, args)
				}
			}
			_, args := base.MustBuild(ctx, nil)
			if !slices.Equal(args, []any{1}) {
				t.Fatalf("base filter changed: %v", args)
			}
		})
	}
}

// Same work as the SELECT POC: two independently built and rendered pages.
func BenchmarkSelectBranchPair(b *testing.B) {
	filters := make([]bob.Mod[dialect.SelectEditor], 1, 4)
	filters[0] = sm.From(Quote("jobs"))
	for i, col := range []string{"tenant_id", "queue_id", "active"} {
		filters = append(filters, sm.Where(Quote(col).EQ(Arg(i+1))))
	}
	base := Select(filters...)
	ctx := b.Context()
	b.Run("apply", func(b *testing.B) {
		for b.Loop() {
			for _, id := range [...]int{11, 22} {
				q := base.WithMods(sm.Where(Quote("owner_id").EQ(Arg(id))), sm.OrderBy(Quote("id")), sm.Limit(50))
				q.MustBuild(ctx, nil)
			}
		}
	})
}
