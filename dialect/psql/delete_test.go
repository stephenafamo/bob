package psql_test

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/fm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestDelete(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: psql.Delete(
				dm.From("films"),
				dm.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = $1)`,
			ExpectedArgs: []any{"Drama"},
		},
		"with using": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with multiple using items": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.Using("departments"),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts, departments
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"using item with inner join": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts",
					dm.InnerJoin("departments").OnEQ(
						psql.Quote("accounts", "dept_id"),
						psql.Quote("departments", "id"),
					),
				),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  INNER JOIN departments ON (accounts.dept_id = departments.id)
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"using then standalone inner join": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.InnerJoin("departments").OnEQ(
					psql.Quote("accounts", "dept_id"),
					psql.Quote("departments", "id"),
				),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  INNER JOIN departments ON (accounts.dept_id = departments.id)
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"standalone inner join on last using item": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.Using("departments"),
				dm.InnerJoin("regions").OnEQ(
					psql.Quote("departments", "id"),
					psql.Quote("regions", "dept_id"),
				),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts, (departments
			  INNER JOIN regions ON (departments.id = regions.dept_id))
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"interleaved using and standalone joins": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.InnerJoin("departments").OnEQ(
					psql.Quote("accounts", "dept_id"),
					psql.Quote("departments", "id"),
				),
				dm.Using("regions"),
				dm.LeftJoin("countries").OnEQ(
					psql.Quote("regions", "country_id"),
					psql.Quote("countries", "id"),
				),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING (accounts
			  INNER JOIN departments ON (accounts.dept_id = departments.id)), (regions
			  LEFT JOIN countries ON (regions.country_id = countries.id))
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"using item with left join and alias": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts",
					dm.LeftJoin("departments").OnEQ(
						psql.Quote("accounts", "dept_id"),
						psql.Quote("departments", "id"),
					),
				).As("a"),
				dm.Where(psql.Quote("a", "dept_id").IsNotNull()),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts AS "a"
			  LEFT JOIN departments ON ("accounts"."dept_id" = "departments"."id")
			  WHERE ("a"."dept_id" IS NOT NULL)`,
		},
		"multiple using items with cross join on second": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.Using("departments", dm.CrossJoin("regions")),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts, (departments
			  CROSS JOIN regions)
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"with using function": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using(dm.UsingFunction(psql.F("generate_series", 1, 2)())),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING generate_series(1, 2)
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"with using function rows from": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using(dm.UsingFunction(
					psql.F("generate_series", 1, 1)(),
					psql.F("json_to_recordset", psql.Arg(`[{"a":1}]`))(fm.Columns("a", "INTEGER")),
				)),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING ROWS FROM (generate_series(1, 1), json_to_recordset($1) AS (a INTEGER))
			  WHERE (employees.id = $2)`,
			ExpectedArgs: []any{`[{"a":1}]`, 1},
		},
		"with multiple using items join rows from and table": {
			Query: psql.Delete(
				dm.From("employees"),
				dm.Using("accounts",
					dm.LeftJoin("departments").OnEQ(
						psql.Quote("accounts", "dept_id"),
						psql.Quote("departments", "id"),
					),
				),
				dm.Using(dm.UsingFunction(
					psql.F("generate_series", 1, 1)(),
					psql.F("json_to_recordset", psql.Arg(`[{"a":1}]`))(fm.Columns("a", "INTEGER")),
				)),
				dm.Using("regions"),
				dm.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `DELETE FROM employees USING (accounts
			  LEFT JOIN departments ON ("accounts"."dept_id" = "departments"."id")), ROWS FROM (generate_series(1, 1), json_to_recordset($1) AS (a INTEGER)), regions
			  WHERE (employees.id = $2)`,
			ExpectedArgs: []any{`[{"a":1}]`, 1},
		},
		"where current of": {
			Query: psql.Delete(
				dm.From("films"),
				dm.WhereCurrentOf("c_films"),
			),
			ExpectedSQL: `DELETE FROM films WHERE CURRENT OF c_films`,
		},
		"returning with old alias": {
			Query: psql.Delete(
				dm.From("users"),
				dm.Where(psql.Quote("id").EQ(psql.Arg(42))),
				dm.Returning(
					psql.Quote("before", "id"),
					psql.Quote("before", "primary_email"),
				).WithOldAs("before"),
			),
			ExpectedSQL:  `DELETE FROM users WHERE ("id" = $1) RETURNING WITH (OLD AS "before") "before"."id", "before"."primary_email"`,
			ExpectedArgs: []any{42},
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func TestDeleteWhereCurrentOfConflict(t *testing.T) {
	_, _, err := bob.Build(context.Background(), psql.Delete(
		dm.From("films"),
		dm.Where(psql.Quote("id").EQ(psql.Arg(1))),
		dm.WhereCurrentOf("c_films"),
	))

	if err == nil {
		t.Fatal("expected error when both WHERE and WHERE CURRENT OF are set")
	}
}
