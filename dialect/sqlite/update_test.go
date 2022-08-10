package sqlite_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite"
)

func TestUpdate(t *testing.T) {
	qm := sqlite.UpdateQM
	selectQM := sqlite.SelectQM
	examples := d.Testcases{
		"simple": {
			Query: sqlite.Update(
				qm.Table("films"),
				qm.SetArg("kind", "Dramatic"),
				qm.Where(sqlite.X("kind").EQ(sqlite.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = ?1 WHERE (kind = ?2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: sqlite.Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.From("accounts"),
				qm.Where(sqlite.X("accounts.name").EQ(sqlite.Arg("Acme Corporation"))),
				qm.Where(sqlite.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
			  WHERE (accounts.name = ?1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees AS "e" NOT INDEXED
				SET "sales_count" = sales_count + 1
				WHERE (id = (SELECT sales_person FROM accounts WHERE (name = ?1)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: sqlite.Update(
				qm.TableAs("employees", "e"),
				qm.TableNotIndexed(),
				qm.Set("sales_count", "sales_count + 1"),
				qm.Where(sqlite.X("id").EQ(sqlite.P(sqlite.Select(
					selectQM.Columns("sales_person"),
					selectQM.From("accounts"),
					selectQM.Where(sqlite.X("name").EQ(sqlite.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	d.RunTests(t, examples)
}
