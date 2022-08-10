package mysql_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/mysql"
)

func TestDelete(t *testing.T) {
	qm := mysql.DeleteQM
	examples := d.Testcases{
		"simple": {
			Query: mysql.Delete(
				qm.From("films"),
				qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = ?)`,
			ExpectedArgs: []any{"Drama"},
		},
		"multiple tables": {
			Query: mysql.Delete(
				qm.From("films"),
				qm.From("actors"),
				qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films, actors WHERE (kind = ?)`,
			ExpectedArgs: []any{"Drama"},
		},
		"with limit and offest": {
			Query: mysql.Delete(
				qm.From("films"),
				qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
				qm.Limit(10),
				qm.OrderBy("producer").Desc(),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = ?) ORDER BY producer DESC LIMIT 10`,
			ExpectedArgs: []any{"Drama"},
		},
		"with using": {
			Query: mysql.Delete(
				qm.From("employees"),
				qm.Using("accounts"),
				qm.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
				qm.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = ?)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
	}

	d.RunTests(t, examples)
}
