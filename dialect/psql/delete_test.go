package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
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
	}

	testutils.RunTests(t, examples, formatter)
}

func TestDeleteReturningWith(t *testing.T) {
	examples := testutils.Testcases{
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

	testutils.RunTests(t, examples, nil)
}
