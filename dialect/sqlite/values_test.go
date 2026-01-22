package sqlite_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/dialect/sqlite/vm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestValues(t *testing.T) {
	examples := testutils.Testcases{
		"simple values": {
			Doc:          "Simple values query with some rows",
			ExpectedSQL:  "VALUES (?1, ?2, ?3), (?4, ?5, ?6)",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7},
			Query: sqlite.Values(
				vm.RowValue(sqlite.Arg(1, 2, 3)),
				vm.RowValue(sqlite.Arg(5, 6, 7)),
			),
		},
		"values with nested select": {
			Doc:          "Values query with nested select query as a row item",
			ExpectedSQL:  "VALUES ((SELECT id FROM users LIMIT ?1), ?2), (?3, ?4)",
			ExpectedArgs: []any{1, 2, 98, 99},
			Query: sqlite.Values(
				vm.RowValue(
					sqlite.Select(
						sm.Columns("id"),
						sm.From("users"),
						sm.Limit(sqlite.Arg(1)),
					),
					sqlite.Arg(2),
				),
				vm.RowValue(sqlite.Arg(98), sqlite.Arg(99)),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}
