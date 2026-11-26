package psql

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/mm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/dialect/psql/vm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/bob/orm/genruntime"
	"github.com/stephenafamo/scan"
)

func checkStatement(t *testing.T, ctx context.Context, q bob.Query, want string, wantArgs ...any) {
	t.Helper()
	sql, args, err := bob.BuildN(ctx, nil, q, 3)
	if err != nil || strings.Join(strings.Fields(sql), " ") != want || !slices.Equal(args, wantArgs) {
		t.Fatalf("built %q %v (%v); want %q %v", sql, args, err, want, wantArgs)
	}
}

func TestInsertOwnsTableColumns(t *testing.T) {
	table := NewTable[*User, *UserSetter]("", "users", expr.ColsForStruct[User]("users"), orm.RowMapping[*User, []*User]{})
	before := table.Insert()
	want, _ := before.MustBuild(t.Context(), nil)
	changed := table.Insert(
		bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) { cols := q.ColumnNames(); cols[0] = "changed"; q.SetTableColumns(cols...) }),
	)
	if got, _ := changed.MustBuild(t.Context(), nil); got == want {
		t.Fatal("the edited insert still uses the default columns")
	}
	for _, q := range []bob.Query{before, table.Insert()} {
		if got, _ := bob.MustBuild(t.Context(), nil, q); got != want {
			t.Fatalf("editing one insert changed another: got %q; want %q", got, want)
		}
	}
}

func TestPostgresWriteBranches(t *testing.T) {
	ctx := t.Context()
	t.Run("insert", func(t *testing.T) {
		base := Insert(im.Into("jobs", "id"), im.Values(Arg(1)), im.OnConflict("id").DoNothing())
		sibling := base.WithMods(im.Values(Arg(2)))
		child := sibling.WithMods(bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) {
			q.SetValue(0, 0, Arg(9))
			im.OnConflict("other").DoNothing().Edit(q)
		}))
		checkStatement(t, ctx, child, `INSERT INTO jobs("id") VALUES ($3), ($4) ON CONFLICT (other) DO NOTHING`, 9, 2)
		checkStatement(t, ctx, sibling, `INSERT INTO jobs("id") VALUES ($3), ($4) ON CONFLICT (id) DO NOTHING`, 1, 2)
		checkStatement(t, ctx, base, `INSERT INTO jobs("id") VALUES ($3) ON CONFLICT (id) DO NOTHING`, 1)
	})
	t.Run("update", func(t *testing.T) {
		base := Update(um.Table("jobs"), um.SetCol("id").ToArg(1), um.From("src", um.InnerJoin("owners").Using("id")))
		sibling := base.WithMods(um.Where(Quote("tenant").EQ(Arg(2))))
		child := sibling.WithMods(bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
			um.InnerJoin("extra").Using("other").Edit(q)
			q.SetWhereAt(0, Quote("tenant").EQ(Arg(9)))
		}))
		checkStatement(
			t,
			ctx,
			child,
			`UPDATE jobs SET "id" = $3 FROM src INNER JOIN owners USING ("id") INNER JOIN extra USING ("other") WHERE ("tenant" = $4)`,
			1,
			9,
		)
		checkStatement(t, ctx, sibling, `UPDATE jobs SET "id" = $3 FROM src INNER JOIN owners USING ("id") WHERE ("tenant" = $4)`, 1, 2)
		checkStatement(t, ctx, base, `UPDATE jobs SET "id" = $3 FROM src INNER JOIN owners USING ("id")`, 1)
	})
	t.Run("delete", func(t *testing.T) {
		base := Delete(dm.From("jobs"), dm.Using("src"), dm.Where(Quote("tenant").EQ(Arg(1))))
		sibling := base.WithMods(dm.Returning("id"))
		child := sibling.WithMods(bob.ModFunc[dialect.DeleteEditor](func(q dialect.DeleteEditor) {
			dm.Using("other").Edit(q)
			q.SetWhereAt(0, Quote("tenant").EQ(Arg(9)))
		}))
		checkStatement(t, ctx, child, `DELETE FROM jobs USING src, other WHERE ("tenant" = $3) RETURNING id`, 9)
		checkStatement(t, ctx, sibling, `DELETE FROM jobs USING src WHERE ("tenant" = $3) RETURNING id`, 1)
		checkStatement(t, ctx, base, `DELETE FROM jobs USING src WHERE ("tenant" = $3)`, 1)
	})
	t.Run("merge", func(t *testing.T) {
		base := Merge(mm.Into("jobs"), mm.Using("src").On(Raw("true")), mm.WhenNotMatched().ThenInsert(mm.Columns("id"), mm.Values(Arg(1))))
		sibling := base.WithMods(mm.Returning("id"))
		child := sibling.WithMods(bob.ModFunc[dialect.MergeEditor](func(q dialect.MergeEditor) {
			mm.WhenMatched().ThenDelete().Edit(q)
		}))
		const prefix = `MERGE INTO jobs USING src ON true WHEN NOT MATCHED THEN INSERT `
		checkStatement(t, ctx, child, prefix+`("id") VALUES ($3) WHEN MATCHED THEN DELETE RETURNING id`, 1)
		checkStatement(t, ctx, sibling, prefix+`("id") VALUES ($3) RETURNING id`, 1)
		checkStatement(t, bob.SkipQueryHooks(ctx), base, prefix+`("id") VALUES ($3)`, 1)
	})
	t.Run("values", func(t *testing.T) {
		base := Values(vm.RowValue(Arg(1)))
		sibling := base.WithMods(vm.RowValue(Arg(2)), vm.Limit(1))
		child := sibling.WithMods(bob.ModFunc[dialect.ValuesEditor](func(q dialect.ValuesEditor) { q.SetValue(0, 0, Arg(9)) }))
		checkStatement(t, ctx, child, "VALUES ($3), ($4) LIMIT 1", 9, 2)
		checkStatement(t, ctx, sibling, "VALUES ($3), ($4) LIMIT 1", 1, 2)
		checkStatement(t, ctx, base, "VALUES ($3)", 1)
	})
	t.Run("raw", func(t *testing.T) {
		args := []any{1, 2}
		q := RawQuery("VALUES (?, ?)", args...)
		args[1] = 9
		checkStatement(t, ctx, q, "VALUES ($3, $4)", 1, 2)
	})
}

func TestPostgresWriteContextualIsolation(t *testing.T) {
	type key struct{}
	base := Insert(im.Into("jobs"), im.Values(Arg(0)))
	byTenant := base.WithMods(bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) {
		q.AppendContextualModFunc(func(ctx context.Context, q dialect.InsertEditor) (context.Context, error) {
			q.SetValue(0, 0, Arg(ctx.Value(key{})))
			return ctx, nil
		})
	}))
	for _, id := range []int{1, 2, 3, 4} {
		t.Run("build", func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(t.Context(), key{}, id)
			for range 20 {
				checkStatement(t, ctx, byTenant, `INSERT INTO jobs VALUES ($3)`, id)
				checkStatement(t, ctx, base, `INSERT INTO jobs VALUES ($3)`, 0)
			}
		})
	}
}

func TestTableUpdateHookBranches(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE jobs(id int, tenant int, name text);
		INSERT INTO jobs VALUES (1, 10, 'a'), (1, 20, 'b'), (2, 10, 'c')`); err != nil {
		t.Fatal(err)
	}
	table := NewTable[*User, *UserSetter]("", "jobs", expr.NewColumnsExpr("name"), orm.RowMapping[*User, []*User]{})
	table.UpdateQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q UpdateEditor) (context.Context, error) {
		um.Where(Quote("tenant").EQ(Arg(10))).Edit(q)
		return ctx, nil
	})
	base := table.Update(um.SetCol("name").ToArg("ok"))
	left := base.WithMods(um.Where(Quote("id").EQ(Arg(1))))
	right := base.WithMods(um.Where(Quote("id").EQ(Arg(2))))
	for _, q := range []*modelRows[dialect.UpdateEditor, *User, []*User]{left, right, left} {
		rows, err := q.All(ctx, tx)
		if err != nil || len(rows) != 1 || rows[0].Name != "ok" {
			t.Fatalf("updated %v (%v); want one matching row", rows, err)
		}
	}
	const prefix = `UPDATE "jobs" SET "name" = $3`
	checkStatement(t, bob.SkipQueryHooks(ctx), base, prefix+` RETURNING "name"`, "ok")
	checkStatement(t, bob.SkipQueryHooks(ctx), left, prefix+` WHERE ("id" = $4) RETURNING "name"`, "ok", 1)
	checkStatement(t, bob.SkipQueryHooks(ctx), right, prefix+` WHERE ("id" = $4) RETURNING "name"`, "ok", 2)
}

func TestGeneratedPostgresBranches(t *testing.T) {
	ctx := t.Context()
	gen := genruntime.NewModQuery[bob.SliceTransformer[int, []int]](expr.Raw("DELETE FROM jobs RETURNING id"), dialect.Dialect, bob.QueryTypeDelete,
		dialect.NewDeleteSource, func(q dialect.DeleteEditor) { dm.From("jobs").Edit(q); dm.Returning("id").Edit(q) }, scan.SingleColumnMapper[int])
	base := gen.WithMods(dm.Where(Quote("tenant").EQ(Arg(1))))
	child := base.WithMods(dm.Where(Quote("id").EQ(Arg(2))))
	checkStatement(t, ctx, child, `DELETE FROM jobs WHERE ("tenant" = $3) AND ("id" = $4) RETURNING id`, 1, 2)
	checkStatement(t, ctx, base, `DELETE FROM jobs WHERE ("tenant" = $3) RETURNING id`, 1)
	checkStatement(t, ctx, gen, `DELETE FROM jobs RETURNING id`)
}
