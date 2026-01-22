package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/vm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestValues(t *testing.T) {
	examples := testutils.Testcases{
		"simple values": {
			Doc:          "Simple values query with some rows",
			ExpectedSQL:  "VALUES ROW(?, ?, ?), ROW(?, ?, ?)",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7},
			Query: mysql.Values(
				vm.ValueRow(mysql.Arg(1, 2, 3)),
				vm.ValueRow(mysql.Arg(5, 6, 7)),
			),
		},
		"simple limit offset arg": {
			Doc:          "Simple values query with limit and offset as argument",
			ExpectedSQL:  "VALUES ROW(?, ?, ?), ROW(?, ?, ?) LIMIT 10",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7},
			Query: mysql.Values(
				vm.ValueRow(mysql.Arg(1, 2, 3)),
				vm.ValueRow(mysql.Arg(5, 6, 7)),
				vm.Limit(10),
			),
		},
		"values with order by": {
			Doc:          "Simple values query with order by clause",
			ExpectedSQL:  "VALUES ROW(?, ?, ?), ROW(?, ?, ?) ORDER BY column_1 DESC",
			ExpectedArgs: []any{"one", 2, 3, "five", 6, 7},
			Query: mysql.Values(
				vm.ValueRow(mysql.Arg("one", 2, 3)),
				vm.ValueRow(mysql.Arg("five", 6, 7)),
				vm.OrderBy("column_1").Desc(),
			),
		},
		"values with nested select": {
			Doc:          "Values query with nested select query as a row item",
			ExpectedSQL:  "VALUES ((SELECT id FROM users LIMIT 1), ?), (?, ?)",
			ExpectedArgs: []any{2, 98, 99},
			Query: mysql.Values(
				vm.ValueRow(
					mysql.Select(
						sm.Columns("id"),
						sm.From("users"),
						sm.Limit(1),
					),
					mysql.Arg(2),
				),
				vm.ValueRow(mysql.Arg(98), mysql.Arg(99)),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}
