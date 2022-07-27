package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
)

func TestDelete(t *testing.T) {
	var qm = DeleteQM{}
	var examples = d.Testcases{
		"simple": {
			Query: Delete(
				qm.From("films"),
				qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = $1)`,
			ExpectedArgs: []any{"Drama"},
		},
		"with using": {
			Query: Delete(
				qm.From("employees"),
				qm.Using("accounts"),
				qm.Where(qm.X("accounts.name").EQ(qm.Arg("Acme Corporation"))),
				qm.Where(qm.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
	}

	d.RunTests(t, examples)
}
