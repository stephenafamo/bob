package mssql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mssql"
	"github.com/stephenafamo/bob/dialect/mssql/sm"
	"github.com/stephenafamo/bob/dialect/mssql/um"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: mssql.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
				um.Where(mssql.Quote("kind").EQ(mssql.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET [kind] = @p1 WHERE ([kind] = @p2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: mssql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.From("accounts"),
				um.Where(mssql.Quote("accounts", "name").EQ(mssql.Arg("Acme Corporation"))),
				um.Where(mssql.Quote("employees", "id").EQ(mssql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL: `UPDATE employees SET [sales_count] = sales_count + 1 FROM accounts
			  WHERE ([accounts].[name] = @p1)
			  AND ([employees].[id] = [accounts].[sales_person])`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with output": {
			Doc: "Update with OUTPUT clause",
			Query: mssql.Update(
				um.Table("inventory"),
				um.SetCol("quantity").ToArg(0),
				um.Output("deleted.product_id", "deleted.quantity"),
				um.Where(mssql.Quote("quantity").LT(mssql.Arg(5))),
			),
			ExpectedSQL:  `UPDATE inventory SET [quantity] = @p1 OUTPUT deleted.product_id, deleted.quantity WHERE ([quantity] < @p2)`,
			ExpectedArgs: []any{0, 5},
		},
		"with top": {
			Doc: "Update with TOP to limit rows updated",
			Query: mssql.Update(
				um.Table("employees"),
				um.Top(10),
				um.SetCol("bonus").ToArg(500),
			),
			ExpectedSQL:  `UPDATE TOP (10) employees SET [bonus] = @p1`,
			ExpectedArgs: []any{500},
		},
		"with sub-select": {
			ExpectedSQL:  `UPDATE employees SET [sales_count] = sales_count + 1 WHERE ([id] = ((SELECT sales_person FROM accounts WHERE ([name] = @p1))))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: mssql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.Where(mssql.Quote("id").EQ(mssql.Group(mssql.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(mssql.Quote("name").EQ(mssql.Arg("Acme Corporation"))),
				)))),
			),
		},
		"with CTE and output": {
			Query: mssql.Update(
				um.With("old_data").As(mssql.Select(
					sm.Columns("id", "name"),
					sm.From("users"),
					sm.Where(mssql.Quote("active").EQ(mssql.Arg(false))),
				)),
				um.Table("users"),
				um.SetCol("active").ToArg(true),
				um.Output("inserted.id", "inserted.active"),
				um.Where(mssql.Quote("id").In(mssql.Raw("SELECT id FROM old_data"))),
			),
			ExpectedSQL:  `WITH [old_data] AS (SELECT id, name FROM users WHERE ([active] = @p1)) UPDATE users SET [active] = @p2 OUTPUT inserted.id, inserted.active WHERE ([id] IN (SELECT id FROM old_data))`,
			ExpectedArgs: []any{false, true},
		},
		"with join": {
			Query: mssql.Update(
				um.Table("t1"),
				um.SetCol("col1").To(mssql.Quote("t2", "col2")),
				um.From("t2"),
				um.InnerJoin("t3").OnEQ(mssql.Quote("t2", "id"), mssql.Quote("t3", "id")),
				um.Where(mssql.Quote("t1", "id").EQ(mssql.Quote("t2", "ref_id"))),
			),
			ExpectedSQL: `UPDATE t1 SET [col1] = [t2].[col2] FROM t2 INNER JOIN t3 ON ([t2].[id] = [t3].[id]) WHERE ([t1].[id] = [t2].[ref_id])`,
		},
		"with left join": {
			Doc: "Update with LEFT JOIN to set defaults for unmatched rows",
			Query: mssql.Update(
				um.Table("t1"),
				um.SetCol("status").ToArg("orphan"),
				um.From("t1").As("src"),
				um.LeftJoin("t2").As("ref").OnEQ(mssql.Quote("src", "ref_id"), mssql.Quote("ref", "id")),
				um.Where(mssql.Quote("t1", "id").EQ(mssql.Quote("src", "id"))),
				um.Where(mssql.Quote("ref", "id").IsNull()),
			),
			ExpectedSQL: `UPDATE t1 SET [status] = @p1
				FROM t1 AS [src] LEFT JOIN t2 AS [ref] ON ([src].[ref_id] = [ref].[id])
				WHERE ([t1].[id] = [src].[id])
				AND ([ref].[id] IS NULL)`,
			ExpectedArgs: []any{"orphan"},
		},
		"with output and from": {
			Doc: "OUTPUT must appear between SET and FROM",
			Query: mssql.Update(
				um.Table("t1"),
				um.SetCol("col1").To(mssql.Quote("t2", "col2")),
				um.Output("inserted.col1", "deleted.col1"),
				um.From("t2"),
				um.Where(mssql.Quote("t1", "id").EQ(mssql.Quote("t2", "ref_id"))),
			),
			ExpectedSQL: `UPDATE t1 SET [col1] = [t2].[col2]
				OUTPUT inserted.col1, deleted.col1
				FROM t2
				WHERE ([t1].[id] = [t2].[ref_id])`,
		},
	}

	testutils.RunTests(t, examples, nil)
}
