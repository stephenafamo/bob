package psql_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/psql"
)

func TestUpdate(t *testing.T) {
	var qm = psql.UpdateQM{}
	var selectQM = psql.SelectQM{}

	var examples = d.Testcases{
		"simple": {
			Query: psql.Update(
				qm.Table("films"),
				qm.SetArg("kind", "Dramatic"),
				qm.Where(psql.X("kind").EQ(psql.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET kind = $1 WHERE (kind = $2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: psql.Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.From("accounts"),
				qm.Where(psql.X("accounts.name").EQ(psql.Arg("Acme Corporation"))),
				qm.Where(psql.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `UPDATE employees SET sales_count = sales_count + 1 FROM accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees SET sales_count = sales_count + 1 WHERE (id =
				  (SELECT sales_person FROM accounts WHERE (name = $1)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: psql.Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.Where(psql.X("id").EQ(psql.P(psql.Select(
					selectQM.Select("sales_person"),
					selectQM.From("accounts"),
					selectQM.Where(psql.X("name").EQ(psql.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	d.RunTests(t, examples)
}
