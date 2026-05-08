package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: psql.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
				um.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = $1 WHERE (kind = $2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.From("accounts"),
				um.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
				um.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 WHERE (id =
				  (SELECT sales_person FROM accounts WHERE (name = $1)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.Where(psql.Quote("id").EQ(psql.Group(psql.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(psql.Quote("name").EQ(psql.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func TestUpdateReturningWith(t *testing.T) {
	examples := testutils.Testcases{
		"returning with old and new aliases": {
			Query: psql.Update(
				um.Table("users"),
				um.SetCol("primary_email").ToArg("new@example.com"),
				um.Where(psql.Quote("id").EQ(psql.Arg(1))),
				um.Returning(
					psql.Quote("before", "primary_email"),
					psql.Quote("after", "primary_email"),
				).WithOldAs("before").WithNewAs("after"),
			),
			ExpectedSQL:  `UPDATE users SET "primary_email" = $1 WHERE ("id" = $2) RETURNING WITH (OLD AS "before", NEW AS "after") "before"."primary_email", "after"."primary_email"`,
			ExpectedArgs: []any{"new@example.com", 1},
		},
	}

	testutils.RunTests(t, examples, nil)
}
