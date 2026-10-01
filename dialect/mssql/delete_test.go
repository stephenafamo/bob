package mssql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mssql"
	"github.com/stephenafamo/bob/dialect/mssql/dm"
	"github.com/stephenafamo/bob/dialect/mssql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestDelete(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: mssql.Delete(
				dm.From("films"),
				dm.Where(mssql.Quote("kind").EQ(mssql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE ([kind] = @p1)`,
			ExpectedArgs: []any{"Drama"},
		},
		"with output": {
			Doc: "Delete with OUTPUT clause to capture deleted rows",
			Query: mssql.Delete(
				dm.From("employees"),
				dm.Output("deleted.id", "deleted.name"),
				dm.Where(mssql.Quote("terminated").EQ(mssql.Arg(true))),
			),
			ExpectedSQL:  `DELETE FROM employees OUTPUT deleted.id, deleted.name WHERE ([terminated] = @p1)`,
			ExpectedArgs: []any{true},
		},
		"with top": {
			Doc: "Delete with TOP to limit number of rows deleted",
			Query: mssql.Delete(
				dm.From("logs"),
				dm.Top(1000),
				dm.Where(mssql.Quote("created_at").LT(mssql.Arg("2020-01-01"))),
			),
			ExpectedSQL:  `DELETE TOP (1000) FROM logs WHERE ([created_at] < @p1)`,
			ExpectedArgs: []any{"2020-01-01"},
		},
		"with using join": {
			Doc: "Delete with FROM/JOIN for multi-table delete",
			Query: mssql.Delete(
				dm.From("order_items"),
				dm.Using("orders"),
				dm.Where(mssql.Quote("order_items", "order_id").EQ(mssql.Quote("orders", "id"))),
				dm.Where(mssql.Quote("orders", "status").EQ(mssql.Arg("cancelled"))),
			),
			ExpectedSQL: `DELETE FROM order_items FROM orders
			  WHERE ([order_items].[order_id] = [orders].[id])
			  AND ([orders].[status] = @p1)`,
			ExpectedArgs: []any{"cancelled"},
		},
		"with output and using join": {
			Doc: "Delete with OUTPUT positioned between target table and second FROM",
			Query: mssql.Delete(
				dm.From("order_items"),
				dm.Output("deleted.id", "deleted.order_id"),
				dm.Using("orders"),
				dm.Where(mssql.Quote("order_items", "order_id").EQ(mssql.Quote("orders", "id"))),
				dm.Where(mssql.Quote("orders", "status").EQ(mssql.Arg("cancelled"))),
			),
			ExpectedSQL: `DELETE FROM order_items
				OUTPUT deleted.id, deleted.order_id
				FROM orders
				WHERE ([order_items].[order_id] = [orders].[id])
				AND ([orders].[status] = @p1)`,
			ExpectedArgs: []any{"cancelled"},
		},
		"with inner join": {
			Doc: "Delete with explicit INNER JOIN via second FROM clause",
			Query: mssql.Delete(
				dm.From("order_items"),
				dm.Using("orders"),
				dm.InnerJoin("products").OnEQ(mssql.Quote("orders", "product_id"), mssql.Quote("products", "id")),
				dm.Where(mssql.Quote("order_items", "order_id").EQ(mssql.Quote("orders", "id"))),
				dm.Where(mssql.Quote("products", "discontinued").EQ(mssql.Arg(true))),
			),
			ExpectedSQL: `DELETE FROM order_items
				FROM orders INNER JOIN products ON ([orders].[product_id] = [products].[id])
				WHERE ([order_items].[order_id] = [orders].[id])
				AND ([products].[discontinued] = @p1)`,
			ExpectedArgs: []any{true},
		},
		"with left join": {
			Doc: "Delete with LEFT JOIN to find unmatched rows",
			Query: mssql.Delete(
				dm.From("users"),
				dm.Using("users").As("u"),
				dm.LeftJoin("orders").As("o").OnEQ(mssql.Quote("u", "id"), mssql.Quote("o", "user_id")),
				dm.Where(mssql.Quote("users", "id").EQ(mssql.Quote("u", "id"))),
				dm.Where(mssql.Quote("o", "id").IsNull()),
			),
			ExpectedSQL: `DELETE FROM users
				FROM users AS [u] LEFT JOIN orders AS [o] ON ([u].[id] = [o].[user_id])
				WHERE ([users].[id] = [u].[id])
				AND ([o].[id] IS NULL)`,
		},
		"with CTE": {
			Query: mssql.Delete(
				dm.With("old_users").As(mssql.Select(
					sm.Columns("id"),
					sm.From("users"),
					sm.Where(mssql.Quote("active").EQ(mssql.Arg(false))),
				)),
				dm.From("users"),
				dm.Where(mssql.Quote("id").In(mssql.Raw("SELECT id FROM old_users"))),
			),
			ExpectedSQL:  `WITH [old_users] AS (SELECT id FROM users WHERE ([active] = @p1)) DELETE FROM users WHERE ([id] IN (SELECT id FROM old_users))`,
			ExpectedArgs: []any{false},
		},
	}

	testutils.RunTests(t, examples, nil)
}
