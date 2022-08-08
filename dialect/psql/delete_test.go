package psql_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/psql"
)

func TestDelete(t *testing.T) {
	qm := psql.DeleteQM
	examples := d.Testcases{
		"simple": {
			Query: psql.Delete(
				qm.From("films"),
				qm.Where(psql.X("kind").EQ(psql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = $1)`,
			ExpectedArgs: []any{"Drama"},
		},
		"with using": {
			Query: psql.Delete(
				qm.From("employees"),
				qm.Using("accounts"),
				qm.Where(psql.X("accounts.name").EQ(psql.Arg("Acme Corporation"))),
				qm.Where(psql.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
	}

	d.RunTests(t, examples)
}
