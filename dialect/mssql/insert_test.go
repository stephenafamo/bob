package mssql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mssql"
	"github.com/stephenafamo/bob/dialect/mssql/im"
	"github.com/stephenafamo/bob/dialect/mssql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestInsert(t *testing.T) {
	examples := testutils.Testcases{
		"simple insert": {
			Query: mssql.Insert(
				im.Into("films"),
				im.Values(mssql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (@p1, @p2, @p3, @p4, @p5, @p6)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"insert with columns": {
			Query: mssql.Insert(
				im.Into("films", "code", "title", "len"),
				im.Values(mssql.Arg("UA502", "Bananas", 105)),
			),
			ExpectedSQL:  `INSERT INTO films ([code], [title], [len]) VALUES (@p1, @p2, @p3)`,
			ExpectedArgs: []any{"UA502", "Bananas", 105},
		},
		"insert from select": {
			Query: mssql.Insert(
				im.Into("films"),
				im.Query(mssql.Select(
					sm.From("tmp_films"),
					sm.Where(mssql.Quote("date_prod").LT(mssql.Arg("1971-07-13"))),
				)),
			),
			ExpectedSQL:  `INSERT INTO films SELECT * FROM tmp_films WHERE ([date_prod] < @p1)`,
			ExpectedArgs: []any{"1971-07-13"},
		},
		"bulk insert": {
			Query: mssql.Insert(
				im.Into("films"),
				im.Values(mssql.Arg("UA502", "Bananas", 105)),
				im.Values(mssql.Arg("UA503", "Annie Hall", 93)),
			),
			ExpectedSQL: `INSERT INTO films VALUES
				(@p1, @p2, @p3),
				(@p4, @p5, @p6)`,
			ExpectedArgs: []any{
				"UA502", "Bananas", 105,
				"UA503", "Annie Hall", 93,
			},
		},
		"insert with output": {
			Doc: "Insert with OUTPUT clause (T-SQL equivalent of RETURNING)",
			Query: mssql.Insert(
				im.Into("films", "title", "len"),
				im.Output("inserted.id", "inserted.title"),
				im.Values(mssql.Arg("Bananas", 105)),
			),
			ExpectedSQL:  `INSERT INTO films ([title], [len]) OUTPUT inserted.id, inserted.title VALUES (@p1, @p2)`,
			ExpectedArgs: []any{"Bananas", 105},
		},
		"insert with top": {
			Doc: "Insert with TOP to limit rows inserted from a select",
			Query: mssql.Insert(
				im.Into("target_table"),
				im.Top(100),
				im.Query(mssql.Select(
					sm.From("source_table"),
				)),
			),
			ExpectedSQL: `INSERT TOP (100) INTO target_table SELECT * FROM source_table`,
		},
		"insert with CTE": {
			Query: mssql.Insert(
				im.With("new_films").As(mssql.Select(
					sm.Columns("title", "len"),
					sm.From("tmp_films"),
					sm.Where(mssql.Quote("len").GT(mssql.Arg(120))),
				)),
				im.Into("films", "title", "len"),
				im.Query(mssql.Select(
					sm.From("new_films"),
				)),
			),
			ExpectedSQL:  `WITH [new_films] AS (SELECT title, len FROM tmp_films WHERE ([len] > @p1)) INSERT INTO films ([title], [len]) SELECT * FROM new_films`,
			ExpectedArgs: []any{120},
		},
	}

	testutils.RunTests(t, examples, nil)
}
