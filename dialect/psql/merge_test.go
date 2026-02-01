package psql_test

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/mm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestMerge(t *testing.T) {
	examples := testutils.Testcases{
		"simple merge with update and insert": {
			Query: psql.Merge(
				mm.Into("customer_account"),
				mm.Using("recent_transactions").As("t").On(
					psql.Quote("t", "customer_id").EQ(psql.Quote("customer_account", "customer_id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("balance").ToExpr(
							psql.Raw("balance + ?", psql.Quote("t", "transaction_value")),
						),
					),
				),
				mm.WhenNotMatched(
					mm.ThenInsert(
						mm.Columns("customer_id", "balance"),
						mm.Values(psql.Quote("t", "customer_id"), psql.Quote("t", "transaction_value")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO customer_account
				USING recent_transactions AS "t" ON "t"."customer_id" = "customer_account"."customer_id"
				WHEN MATCHED THEN UPDATE SET "balance" = balance + "t"."transaction_value"
				WHEN NOT MATCHED THEN INSERT ("customer_id", "balance") VALUES ("t"."customer_id", "t"."transaction_value")`,
		},
		"merge with condition": {
			Query: psql.Merge(
				mm.Into("wines"),
				mm.Using("wine_stock_changes").As("s").On(
					psql.Quote("s", "winename").EQ(psql.Quote("wines", "winename")),
				),
				mm.WhenNotMatched(
					mm.And(psql.Quote("s", "stock_delta").GT(psql.Arg(0))),
					mm.ThenInsert(
						mm.Values(psql.Quote("s", "winename"), psql.Quote("s", "stock_delta")),
					),
				),
				mm.WhenMatched(
					mm.And(psql.Raw("w.stock + s.stock_delta > 0")),
					mm.ThenUpdate(
						mm.SetCol("stock").ToExpr(psql.Raw("w.stock + s.stock_delta")),
					),
				),
				mm.WhenMatched(
					mm.ThenDelete(),
				),
			),
			ExpectedSQL: `MERGE INTO wines
				USING wine_stock_changes AS "s" ON "s"."winename" = "wines"."winename"
				WHEN NOT MATCHED AND "s"."stock_delta" > $1 THEN INSERT VALUES ("s"."winename", "s"."stock_delta")
				WHEN MATCHED AND w.stock + s.stock_delta > 0 THEN UPDATE SET "stock" = w.stock + s.stock_delta
				WHEN MATCHED THEN DELETE`,
			ExpectedArgs: []any{0},
		},
		"merge with do nothing": {
			Query: psql.Merge(
				mm.Into("target"),
				mm.Using("source").As("s").On(
					psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
				),
				mm.WhenMatched(
					mm.ThenDoNothing(),
				),
				mm.WhenNotMatched(
					mm.ThenDoNothing(),
				),
			),
			ExpectedSQL: `MERGE INTO target
				USING source AS "s" ON "s"."id" = "target"."id"
				WHEN MATCHED THEN DO NOTHING
				WHEN NOT MATCHED THEN DO NOTHING`,
		},
		"merge with target alias": {
			Query: psql.Merge(
				mm.IntoAs("wines", "w"),
				mm.Using("new_wine_list").As("s").On(
					psql.Quote("s", "winename").EQ(psql.Quote("w", "winename")),
				),
				mm.WhenNotMatchedByTarget(
					mm.ThenInsert(
						mm.Values(psql.Quote("s", "winename"), psql.Quote("s", "stock")),
					),
				),
				mm.WhenMatched(
					mm.And(psql.Quote("w", "stock").NE(psql.Quote("s", "stock"))),
					mm.ThenUpdate(
						mm.SetCol("stock").ToExpr(psql.Quote("s", "stock")),
					),
				),
				mm.WhenNotMatchedBySource(
					mm.ThenDelete(),
				),
			),
			ExpectedSQL: `MERGE INTO wines AS "w"
				USING new_wine_list AS "s" ON "s"."winename" = "w"."winename"
				WHEN NOT MATCHED BY TARGET THEN INSERT VALUES ("s"."winename", "s"."stock")
				WHEN MATCHED AND "w"."stock" <> "s"."stock" THEN UPDATE SET "stock" = "s"."stock"
				WHEN NOT MATCHED BY SOURCE THEN DELETE`,
		},
		"merge with returning": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("product_updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("price").ToExpr(psql.Quote("u", "price")),
					),
				),
				mm.WhenNotMatched(
					mm.ThenInsert(
						mm.Columns("id", "name", "price"),
						mm.Values(psql.Quote("u", "id"), psql.Quote("u", "name"), psql.Quote("u", "price")),
					),
				),
				mm.Returning("*"),
			),
			ExpectedSQL: `MERGE INTO products
				USING product_updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET "price" = "u"."price"
				WHEN NOT MATCHED THEN INSERT ("id", "name", "price") VALUES ("u"."id", "u"."name", "u"."price")
				RETURNING *`,
		},
		"merge with subquery as source": {
			Query: psql.Merge(
				mm.Into("target_table"),
				mm.UsingQuery(psql.Select()).As("src").On(
					psql.Quote("src", "id").EQ(psql.Quote("target_table", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("value").ToExpr(psql.Quote("src", "value")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO target_table
				USING (SELECT *) AS "src" ON "src"."id" = "target_table"."id"
				WHEN MATCHED THEN UPDATE SET "value" = "src"."value"`,
		},
		"merge with multiple SetCol from source": {
			Query: psql.Merge(
				mm.Into("employees"),
				mm.Using("employee_updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("employees", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("name").ToExpr(psql.Quote("u", "name")),
						mm.SetCol("salary").ToExpr(psql.Quote("u", "salary")),
						mm.SetCol("department").ToExpr(psql.Quote("u", "department")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO employees
				USING employee_updates AS "u" ON "u"."id" = "employees"."id"
				WHEN MATCHED THEN UPDATE SET "name" = "u"."name", "salary" = "u"."salary", "department" = "u"."department"`,
		},
		"merge with SetCol ToDefault": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("updated_at").ToDefault(),
						mm.SetCol("name").ToExpr(psql.Quote("u", "name")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET "updated_at" = DEFAULT, "name" = "u"."name"`,
		},
		"merge with SetCols ToRow": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCols("name", "price").ToRow(
							psql.Quote("u", "name"),
							psql.Quote("u", "price"),
						),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET ("name", "price") = ROW ("u"."name", "u"."price")`,
		},
		"merge with SetCols ToQuery": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCols("name", "price").ToQuery(
							psql.Select(
								sm.Columns("name", "price"),
								sm.From("default_values"),
								sm.Where(psql.Quote("category").EQ(psql.Quote("u", "category"))),
							),
						),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET ("name", "price") = (SELECT "name", "price" FROM default_values WHERE (category = u.category))`,
		},
		"merge with CTE": {
			Query: psql.Merge(
				mm.With("source_data").As(psql.Select(
					sm.Columns("id", "value"),
					sm.From("temp_table"),
				)),
				mm.Into("target"),
				mm.Using("source_data").As("s").On(
					psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("value").ToExpr(psql.Quote("s", "value")),
					),
				),
			),
			ExpectedSQL: `WITH source_data AS (SELECT "id", "value" FROM temp_table)
				MERGE INTO target
				USING source_data AS "s" ON "s"."id" = "target"."id"
				WHEN MATCHED THEN UPDATE SET "value" = "s"."value"`,
		},
		"merge with INSERT DEFAULT VALUES": {
			Query: psql.Merge(
				mm.Into("audit_log"),
				mm.Using("events").As("e").On(
					psql.Quote("e", "id").EQ(psql.Quote("audit_log", "event_id")),
				),
				mm.WhenNotMatched(
					mm.ThenInsertDefaultValues(),
				),
			),
			ExpectedSQL: `MERGE INTO audit_log
				USING events AS "e" ON "e"."id" = "audit_log"."event_id"
				WHEN NOT MATCHED THEN INSERT DEFAULT VALUES`,
		},
		"merge with OVERRIDING SYSTEM VALUE": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("new_products").As("n").On(
					psql.Quote("n", "sku").EQ(psql.Quote("products", "sku")),
				),
				mm.WhenNotMatched(
					mm.ThenInsert(
						mm.Columns("id", "sku", "name"),
						mm.OverridingSystem(),
						mm.Values(psql.Quote("n", "id"), psql.Quote("n", "sku"), psql.Quote("n", "name")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING new_products AS "n" ON "n"."sku" = "products"."sku"
				WHEN NOT MATCHED THEN INSERT ("id", "sku", "name") OVERRIDING SYSTEM VALUE VALUES ("n"."id", "n"."sku", "n"."name")`,
		},
		"merge with OVERRIDING USER VALUE": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("new_products").As("n").On(
					psql.Quote("n", "sku").EQ(psql.Quote("products", "sku")),
				),
				mm.WhenNotMatched(
					mm.ThenInsert(
						mm.Columns("id", "sku", "name"),
						mm.OverridingUser(),
						mm.Values(psql.Quote("n", "id"), psql.Quote("n", "sku"), psql.Quote("n", "name")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING new_products AS "n" ON "n"."sku" = "products"."sku"
				WHEN NOT MATCHED THEN INSERT ("id", "sku", "name") OVERRIDING USER VALUE VALUES ("n"."id", "n"."sku", "n"."name")`,
		},
		"merge with Recursive CTE": {
			Query: psql.Merge(
				mm.With("hierarchy").As(psql.Select(
					sm.Columns("id", "parent_id", "name"),
					sm.From("categories"),
				)),
				mm.Recursive(true),
				mm.Into("target"),
				mm.Using("hierarchy").As("h").On(
					psql.Quote("h", "id").EQ(psql.Quote("target", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("name").ToExpr(psql.Quote("h", "name")),
					),
				),
			),
			ExpectedSQL: `WITH RECURSIVE hierarchy AS (SELECT "id", "parent_id", "name" FROM categories)
				MERGE INTO target
				USING hierarchy AS "h" ON "h"."id" = "target"."id"
				WHEN MATCHED THEN UPDATE SET "name" = "h"."name"`,
		},
		"merge with Only target": {
			Query: psql.Merge(
				mm.Into("parent_table"),
				mm.Only(),
				mm.Using("source").As("s").On(
					psql.Quote("s", "id").EQ(psql.Quote("parent_table", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("value").ToExpr(psql.Quote("s", "value")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO ONLY parent_table
				USING source AS "s" ON "s"."id" = "parent_table"."id"
				WHEN MATCHED THEN UPDATE SET "value" = "s"."value"`,
		},
		"merge with Only source": {
			Query: psql.Merge(
				mm.Into("target"),
				mm.Using("parent_source").Only().As("s").On(
					psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
				),
				mm.WhenMatched(
					mm.ThenDelete(),
				),
			),
			ExpectedSQL: `MERGE INTO target
				USING ONLY parent_source AS "s" ON "s"."id" = "target"."id"
				WHEN MATCHED THEN DELETE`,
		},
		"merge with OnEQ shortcut": {
			Query: psql.Merge(
				mm.Into("target"),
				mm.Using("source").As("s").OnEQ(
					psql.Quote("s", "id"),
					psql.Quote("target", "id"),
				),
				mm.WhenMatched(
					mm.ThenDoNothing(),
				),
			),
			ExpectedSQL: `MERGE INTO target
				USING source AS "s" ON "s"."id" = "target"."id"
				WHEN MATCHED THEN DO NOTHING`,
		},
		"merge with SetCol To raw value": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("status").To(psql.Raw("'active'")),
						mm.SetCol("counter").To(psql.Raw("counter + 1")),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET "status" = 'active', "counter" = counter + 1`,
		},
		"merge with SetCol ToArg": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("status").ToArg("active"),
						mm.SetCol("quantity").ToArg(100),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET "status" = $1, "quantity" = $2`,
			ExpectedArgs: []any{"active", 100},
		},
		"merge with Set raw expressions": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.Set(
							psql.Raw(`"name" = "u"."name"`),
							psql.Raw(`"price" = "u"."price" * 1.1`),
						),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET "name" = "u"."name", "price" = "u"."price" * 1.1`,
		},
		"merge with SetCols ToExprs without ROW": {
			Query: psql.Merge(
				mm.Into("products"),
				mm.Using("updates").As("u").On(
					psql.Quote("u", "id").EQ(psql.Quote("products", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCols("name", "price").ToExprs(
							psql.Quote("u", "name"),
							psql.Quote("u", "price"),
						),
					),
				),
			),
			ExpectedSQL: `MERGE INTO products
				USING updates AS "u" ON "u"."id" = "products"."id"
				WHEN MATCHED THEN UPDATE SET ("name", "price") = ("u"."name", "u"."price")`,
		},
		"merge with multiple WHEN clauses and conditions": {
			Query: psql.Merge(
				mm.Into("inventory"),
				mm.Using("stock_updates").As("s").On(
					psql.Quote("s", "product_id").EQ(psql.Quote("inventory", "product_id")),
				),
				mm.WhenMatched(
					mm.And(psql.Quote("s", "quantity").EQ(psql.Arg(0))),
					mm.ThenDelete(),
				),
				mm.WhenMatched(
					mm.And(psql.Quote("s", "quantity").GT(psql.Arg(0))),
					mm.ThenUpdate(
						mm.SetCol("quantity").ToExpr(psql.Quote("s", "quantity")),
						mm.SetCol("updated_at").ToDefault(),
					),
				),
				mm.WhenNotMatchedByTarget(
					mm.And(psql.Quote("s", "quantity").GT(psql.Arg(0))),
					mm.ThenInsert(
						mm.Columns("product_id", "quantity"),
						mm.Values(psql.Quote("s", "product_id"), psql.Quote("s", "quantity")),
					),
				),
				mm.WhenNotMatchedBySource(
					mm.ThenUpdate(
						mm.SetCol("quantity").ToArg(0),
					),
				),
				mm.Returning("*"),
			),
			ExpectedSQL: `MERGE INTO inventory
				USING stock_updates AS "s" ON "s"."product_id" = "inventory"."product_id"
				WHEN MATCHED AND "s"."quantity" = $1 THEN DELETE
				WHEN MATCHED AND "s"."quantity" > $2 THEN UPDATE SET "quantity" = "s"."quantity", "updated_at" = DEFAULT
				WHEN NOT MATCHED BY TARGET AND "s"."quantity" > $3 THEN INSERT ("product_id", "quantity") VALUES ("s"."product_id", "s"."quantity")
				WHEN NOT MATCHED BY SOURCE THEN UPDATE SET "quantity" = $4
				RETURNING *`,
			ExpectedArgs: []any{0, 0, 0, 0},
		},
		"merge with CTE columns": {
			Query: psql.Merge(
				mm.With("source_data", "id", "name", "value").As(psql.Select(
					sm.Columns("product_id", "product_name", "price"),
					sm.From("products"),
				)),
				mm.Into("target"),
				mm.Using("source_data").As("s").On(
					psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
				),
				mm.WhenMatched(
					mm.ThenUpdate(
						mm.SetCol("name").ToExpr(psql.Quote("s", "name")),
						mm.SetCol("value").ToExpr(psql.Quote("s", "value")),
					),
				),
			),
			ExpectedSQL: `WITH source_data ("id", "name", "value") AS (SELECT "product_id", "product_name", "price" FROM products)
				MERGE INTO target
				USING source_data AS "s" ON "s"."id" = "target"."id"
				WHEN MATCHED THEN UPDATE SET "name" = "s"."name", "value" = "s"."value"`,
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func TestMergeWithVersion(t *testing.T) {
	t.Run("version 17+ adds RETURNING automatically with mm.Returning", func(t *testing.T) {
		ctx := context.Background()
		ctx = psql.SetVersion(ctx, 17)

		q := psql.Merge(
			mm.Into("target"),
			mm.Using("source").As("s").On(
				psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
			),
			mm.WhenMatched(
				mm.ThenUpdate(
					mm.SetCol("name").ToExpr(psql.Quote("s", "name")),
				),
			),
			mm.Returning("*"),
		)

		sql, args, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		expectedSQL := `MERGE INTO target USING source AS "s" ON "s"."id" = "target"."id" WHEN MATCHED THEN UPDATE SET "name" = "s"."name" RETURNING *`
		diff, err := testutils.QueryDiff(expectedSQL, sql, formatter)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if diff != "" {
			t.Errorf("SQL mismatch:\n%s\nGot: %s", diff, sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})

	t.Run("version 16 does not affect MERGE with explicit RETURNING", func(t *testing.T) {
		ctx := context.Background()
		ctx = psql.SetVersion(ctx, 16)

		q := psql.Merge(
			mm.Into("target"),
			mm.Using("source").As("s").On(
				psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
			),
			mm.WhenMatched(
				mm.ThenUpdate(
					mm.SetCol("name").ToExpr(psql.Quote("s", "name")),
				),
			),
			mm.Returning("*"),
		)

		sql, args, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		// RETURNING should still be present because it was explicitly added
		expectedSQL := `MERGE INTO target USING source AS "s" ON "s"."id" = "target"."id" WHEN MATCHED THEN UPDATE SET "name" = "s"."name" RETURNING *`
		diff, err := testutils.QueryDiff(expectedSQL, sql, formatter)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if diff != "" {
			t.Errorf("SQL mismatch:\n%s\nGot: %s", diff, sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})

	t.Run("no version set - MERGE without RETURNING", func(t *testing.T) {
		ctx := context.Background()
		// No version set

		q := psql.Merge(
			mm.Into("target"),
			mm.Using("source").As("s").On(
				psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
			),
			mm.WhenMatched(
				mm.ThenUpdate(
					mm.SetCol("name").ToExpr(psql.Quote("s", "name")),
				),
			),
		)

		sql, args, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		// No RETURNING because no version set
		expectedSQL := `MERGE INTO target USING source AS "s" ON "s"."id" = "target"."id" WHEN MATCHED THEN UPDATE SET "name" = "s"."name"`
		diff, err := testutils.QueryDiff(expectedSQL, sql, formatter)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if diff != "" {
			t.Errorf("SQL mismatch:\n%s\nGot: %s", diff, sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})

	t.Run("version 17+ with WhenNotMatchedBySource (PG17 feature)", func(t *testing.T) {
		ctx := context.Background()
		ctx = psql.SetVersion(ctx, 17)

		q := psql.Merge(
			mm.Into("target"),
			mm.Using("source").As("s").On(
				psql.Quote("s", "id").EQ(psql.Quote("target", "id")),
			),
			mm.WhenMatched(
				mm.ThenUpdate(
					mm.SetCol("name").ToExpr(psql.Quote("s", "name")),
				),
			),
			mm.WhenNotMatchedBySource(
				mm.ThenDelete(),
			),
			mm.Returning(psql.Quote("target", "id")),
		)

		sql, args, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		expectedSQL := `MERGE INTO target USING source AS "s" ON "s"."id" = "target"."id" WHEN MATCHED THEN UPDATE SET "name" = "s"."name" WHEN NOT MATCHED BY SOURCE THEN DELETE RETURNING "target"."id"`
		diff, err := testutils.QueryDiff(expectedSQL, sql, formatter)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if diff != "" {
			t.Errorf("SQL mismatch:\n%s\nGot: %s", diff, sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})
}
