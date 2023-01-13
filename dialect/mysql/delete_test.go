package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dm"
	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestDelete(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: mysql.Delete(
				dm.From("films"),
				dm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = ?)`,
			ExpectedArgs: []any{"Drama"},
		},
		"multiple tables": {
			Query: mysql.Delete(
				dm.From("films"),
				dm.From("actors"),
				dm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films, actors WHERE (kind = ?)`,
			ExpectedArgs: []any{"Drama"},
		},
		"with limit and offest": {
			Query: mysql.Delete(
				dm.From("films"),
				dm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
				dm.Limit(10),
				dm.OrderBy("producer").Desc(),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = ?) ORDER BY producer DESC LIMIT 10`,
			ExpectedArgs: []any{"Drama"},
		},
		"with using": {
			Query: mysql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
				dm.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = ?)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
	}

	// Does not understand multiple tables syntax
	testutils.RunTests(t, examples, nil)
}
