package mssql_test

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob/dialect/mssql"
	"github.com/stephenafamo/bob/dialect/mssql/vm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestValues(t *testing.T) {
	examples := testutils.Testcases{
		"simple values": {
			Doc:          "Simple values query with some rows",
			ExpectedSQL:  "VALUES (@p1, @p2, @p3), (@p4, @p5, @p6)",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7},
			Query: mssql.Values(
				vm.RowValue(mssql.Arg(1, 2, 3)),
				vm.RowValue(mssql.Arg(5, 6, 7)),
			),
		},
		"values with order by": {
			Doc:          "Simple values query with order by clause",
			ExpectedSQL:  "VALUES (@p1, @p2, @p3), (@p4, @p5, @p6) ORDER BY column1 DESC",
			ExpectedArgs: []any{"one", 2, 3, "five", 6, 7},
			Query: mssql.Values(
				vm.RowValue(mssql.Arg("one", 2, 3)),
				vm.RowValue(mssql.Arg("five", 6, 7)),
				vm.OrderBy("column1").Desc(),
			),
		},
		"values with offset fetch": {
			Doc:          "Values query with OFFSET/FETCH NEXT",
			ExpectedSQL:  "VALUES (@p1, @p2, @p3), (@p4, @p5, @p6) ORDER BY 1 OFFSET @p7 ROWS FETCH NEXT @p8 ROWS ONLY",
			ExpectedArgs: []any{1, 2, 3, 5, 6, 7, 10, 5},
			Query: mssql.Values(
				vm.RowValue(mssql.Arg(1, 2, 3)),
				vm.RowValue(mssql.Arg(5, 6, 7)),
				vm.OrderBy(1),
				vm.Offset(mssql.Arg(10)),
				vm.Fetch(mssql.Arg(5)),
			),
		},
	}

	testutils.RunTests(t, examples, nil)
}

func TestValuesEmptyRows(t *testing.T) {
	q := mssql.Values()
	_, _, err := q.Build(context.Background())
	if err == nil {
		t.Fatal("expected error for VALUES with no rows, got nil")
	}
}
