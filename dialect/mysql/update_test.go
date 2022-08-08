package mysql_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/mysql"
)

func TestUpdate(t *testing.T) {
	qm := mysql.UpdateQM{}
	selectQM := mysql.SelectQM{}
	examples := d.Testcases{
		"simple": {
			Query: mysql.Update(
				qm.Table("films"),
				qm.SetArg("kind", "Dramatic"),
				qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET ` + "`kind`" + ` = ? WHERE (kind = ?)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"update multiple tables": {
			Query: mysql.Update(
				qm.Table("employees, accounts"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
				qm.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `UPDATE employees, accounts
			  SET ` + "`sales_count`" + ` = sales_count + 1 
			  WHERE (accounts.name = ?)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees SET ` + "`sales_count`" + ` = sales_count + 1 WHERE (id =
				  (SELECT sales_person FROM accounts WHERE (name = ?)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: mysql.Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.Where(mysql.X("id").EQ(mysql.P(mysql.Select(
					selectQM.Columns("sales_person"),
					selectQM.From("accounts"),
					selectQM.Where(mysql.X("name").EQ(mysql.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	d.RunTests(t, examples)
}
