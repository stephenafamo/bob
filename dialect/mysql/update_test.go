package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/um"
	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: mysql.Update(
				um.Table("films"),
				um.SetArg("kind", "Dramatic"),
				um.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET ` + "`kind`" + ` = ? WHERE (kind = ?)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"update multiple tables": {
			Query: mysql.Update(
				um.Table("employees, accounts"),
				um.Set("sales_count", "sales_count + 1"),
				um.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
				um.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
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
				um.Table("employees"),
				um.Set("sales_count", "sales_count + 1"),
				um.Where(mysql.X("id").EQ(mysql.P(mysql.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(mysql.X("name").EQ(mysql.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	testutils.RunTests(t, examples, nil)
}
