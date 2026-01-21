package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/vm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestValues(t *testing.T) {
	examples := testutils.Testcases{
		"simple values": {
			Doc:          "Simple values query with some rows",
			ExpectedSQL:  "VALUES ($1,$2,$3), ($4, $5, $6)",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7},
			Query: psql.Values(
				vm.ValueRow(psql.Arg(1, 2, 3)),
				vm.ValueRow(psql.Arg(5, 6, 7)),
			),
		},
		"simple limit offset arg": {
			Doc:          "Simple values query with limit and offset as argument",
			ExpectedSQL:  "VALUES ($1,$2,$3), ($4,$5,$6) LIMIT $7 OFFSET $8",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7, 10, 15},
			Query: psql.Values(
				vm.ValueRow(psql.Arg(1, 2, 3)),
				vm.ValueRow(psql.Arg(5, 6, 7)),
				vm.Offset(psql.Arg(15)),
				vm.Limit(psql.Arg(10)),
			),
		},
		"values with order by": {
			Doc:          "Simple values query with order by clause",
			ExpectedSQL:  "VALUES ($1,$2,$3), ($4,$5,$6) ORDER BY column1 DESC",
			ExpectedArgs: []any{"one", 2, 3, "five", 6, 7},
			Query: psql.Values(
				vm.ValueRow(psql.Arg("one", 2, 3)),
				vm.ValueRow(psql.Arg("five", 6, 7)),
				vm.OrderBy("column1").Desc(),
			),
		},
		"values with fetch": {
			Doc:          "Simple values query with fetch",
			ExpectedSQL:  "VALUES ($1,$2,$3), ($4,$5,$6) OFFSET $7 FETCH FIRST $8 ROWS ONLY",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7, 15, 10},
			Query: psql.Values(
				vm.ValueRow(psql.Arg(1, 2, 3)),
				vm.ValueRow(psql.Arg(5, 6, 7)),
				vm.Offset(psql.Arg(15)),
				vm.Fetch(psql.Arg(10)),
			),
		},
		"values with nested select": {
			Doc:          "Values query with nested select query as a row item",
			ExpectedSQL:  "VALUES ((SELECT id FROM users LIMIT $1), $2), ($3, $4)",
			ExpectedArgs: []any{1, 2, 98, 99},
			Query: psql.Values(
				vm.ValueRow(
					psql.Select(
						sm.Columns("id"),
						sm.From("users"),
						sm.Limit(psql.Arg(1)),
					),
					psql.Arg(2),
				),
				vm.ValueRow(psql.Arg(98), psql.Arg(99)),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}
