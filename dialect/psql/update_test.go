package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
)

func TestUpdate(t *testing.T) {
	var qm = UpdateQM{}
	var selectQM = SelectQM{}

	var examples = d.Testcases{
		"simple": {
			Query: Update(
				qm.Table("films"),
				qm.SetArg("kind", "Dramatic"),
				qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET kind = $1 WHERE (kind = $2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.From("accounts"),
				qm.Where(qm.X("accounts.name").EQ(qm.Arg("Acme Corporation"))),
				qm.Where(qm.X("employees.id").EQ("accounts.sales_person")),
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
			Query: Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.Where(qm.X("id").EQ(qm.P(Select(
					selectQM.Select("sales_person"),
					selectQM.From("accounts"),
					selectQM.Where(qm.X("name").EQ(qm.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	d.RunTests(t, examples)
}
