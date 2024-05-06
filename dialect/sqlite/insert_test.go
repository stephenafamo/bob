package sqlite_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/im"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestInsert(t *testing.T) {
	examples := testutils.Testcases{
		"simple insert": {
			Query: sqlite.Insert(
				im.Into("films"),
				im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"insert from select": {
			Query: sqlite.Insert(
				im.Into("films"),
				im.Query(sqlite.Select(
					sm.From("tmp_films"),
					sm.Where(sqlite.Quote("date_prod").LT(sqlite.Arg("1971-07-13"))),
				)),
			),
			ExpectedSQL:  `INSERT INTO films SELECT * FROM tmp_films WHERE ("date_prod" < ?1)`,
			ExpectedArgs: []any{"1971-07-13"},
		},
		"bulk insert": {
			Query: sqlite.Insert(
				im.Into("films"),
				im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL: `INSERT INTO films VALUES
				(?1, ?2, ?3, ?4, ?5, ?6),
				(?7, ?8, ?9, ?10, ?11, ?12)`,
			ExpectedArgs: []any{
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
			},
		},
		"on conflict do nothing": {
			Query: sqlite.Insert(
				im.Into("films"),
				im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.OnConflict().DoNothing(),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (?1, ?2, ?3, ?4, ?5, ?6) ON CONFLICT DO NOTHING",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"upsert": {
			Query: sqlite.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(sqlite.Arg(8, "Anvil Distribution")),
				im.Values(sqlite.Arg(9, "Sentry Distribution")),
				im.OnConflict("did").DoUpdate(
					im.SetExcluded("dname"),
					im.Where(sqlite.Quote("d", "zipcode").NE(sqlite.S("21201"))),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES (?1, ?2), (?3, ?4)
				ON CONFLICT (did) DO UPDATE
				SET "dname" = EXCLUDED. "dname"
				WHERE ("d"."zipcode" <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"or replace": {
			Query: sqlite.Insert(
				im.OrReplace(),
				im.Into("distributors", "did", "dname"),
				im.Values(sqlite.Arg(8, "Anvil Distribution")),
				im.Values(sqlite.Arg(9, "Sentry Distribution")),
			),
			ExpectedSQL: `INSERT OR REPLACE INTO distributors ("did", "dname")
				VALUES (?1, ?2), (?3, ?4)`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
	}

	testutils.RunTests(t, examples, formatter)
}
