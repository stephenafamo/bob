package sqlite_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/dialect/sqlite/um"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: sqlite.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
				um.Where(sqlite.Quote("kind").EQ(sqlite.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = ?1 WHERE ("kind" = ?2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: sqlite.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.From("accounts"),
				um.Where(sqlite.Quote("accounts", "name").EQ(sqlite.Arg("Acme Corporation"))),
				um.Where(sqlite.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
			  WHERE ("accounts"."name" = ?1)
			  AND ("employees"."id" = "accounts"."sales_person")`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees AS "e" NOT INDEXED
				SET "sales_count" = sales_count + 1
				WHERE ("id" = (SELECT sales_person FROM accounts WHERE ("name" = ?1)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: sqlite.Update(
				um.TableAs("employees", "e"),
				um.NotIndexed(),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.Where(sqlite.Quote("id").EQ(sqlite.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(sqlite.Quote("name").EQ(sqlite.Arg("Acme Corporation"))),
				))),
			),
		},
		"setCol as mod": {
			Query: sqlite.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = ?1`,
			ExpectedArgs: []any{"Dramatic"},
		},
		"setExpr as mod": {
			Query: sqlite.Update(
				um.Table("employees"),
				um.SetExpr(sqlite.Quote("employees", "dept_id")).To(sqlite.Quote("accounts", "dept_id")),
				um.From("accounts"),
				um.Where(sqlite.Quote("employees", "id").EQ(sqlite.Arg(1))),
			),
			ExpectedSQL: `UPDATE employees SET "employees"."dept_id" = "accounts"."dept_id" FROM accounts
			  WHERE ("employees"."id" = ?1)`,
			ExpectedArgs: []any{1},
		},
		"setCol and setExpr via Set helper": {
			Query: sqlite.Update(
				um.Table("employees"),
				um.From("accounts"),
				um.Set(
					um.SetCol("sales_count").To("sales_count + 1"),
					um.SetExpr(sqlite.Quote("employees", "dept_id")).To(sqlite.Quote("accounts", "dept_id")),
				),
				um.Where(sqlite.Quote("employees", "id").EQ(sqlite.Arg(1))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1,
			  "employees"."dept_id" = "accounts"."dept_id" FROM accounts
			  WHERE ("employees"."id" = ?1)`,
			ExpectedArgs: []any{1},
		},
	}

	testutils.RunTests(t, examples, formatter)
}
