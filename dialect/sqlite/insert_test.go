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
				SET "dname" = EXCLUDED."dname"
				WHERE ("d"."zipcode" <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"upsert setCol as mod": {
			Query: sqlite.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(sqlite.Arg(8, "Anvil Distribution")),
				im.OnConflict("did").DoUpdate(
					im.SetCol("dname").To(im.Excluded("dname")),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES (?1, ?2)
				ON CONFLICT (did) DO UPDATE
				SET "dname" = EXCLUDED."dname"`,
			ExpectedArgs: []any{8, "Anvil Distribution"},
		},
		"upsert setCol via Set helper": {
			Query: sqlite.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(sqlite.Arg(8, "Anvil Distribution")),
				im.OnConflict("did").DoUpdate(
					im.Set(
						im.SetCol("dname").To(im.Excluded("dname")),
						im.SetCol("did").To(im.Excluded("did")),
					),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES (?1, ?2)
				ON CONFLICT (did) DO UPDATE
				SET "dname" = EXCLUDED."dname",
				"did" = EXCLUDED."did"`,
			ExpectedArgs: []any{8, "Anvil Distribution"},
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
		"on conflict do update set tuple to exprs": {
			Query: sqlite.Insert(
				im.Into("users", "id", "first_name", "last_name"),
				im.Values(sqlite.Arg(1, "Thomas", "Anderson")),
				im.OnConflict("id").DoUpdate(
					im.SetCols("first_name", "last_name").ToExprs(
						sqlite.Raw("EXCLUDED.first_name"),
						sqlite.Raw("EXCLUDED.last_name"),
					),
				),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "first_name", "last_name") VALUES (?1, ?2, ?3) ON CONFLICT (id) DO UPDATE SET ("first_name", "last_name") = (EXCLUDED.first_name, EXCLUDED.last_name)`,
			ExpectedArgs: []any{1, "Thomas", "Anderson"},
		},
		"on conflict do update set tuple to row": {
			Query: sqlite.Insert(
				im.Into("users", "id", "first_name", "last_name"),
				im.Values(sqlite.Arg(1, "Thomas", "Anderson")),
				im.OnConflict("id").DoUpdate(
					im.SetCols("first_name", "last_name").ToRow(
						sqlite.Arg("Neo"),
						sqlite.Arg("Anderson"),
					),
				),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "first_name", "last_name") VALUES (?1, ?2, ?3) ON CONFLICT (id) DO UPDATE SET ("first_name", "last_name") = (?4, ?5)`,
			ExpectedArgs: []any{1, "Thomas", "Anderson", "Neo", "Anderson"},
		},
		"on conflict do update set tuple to query": {
			Query: sqlite.Insert(
				im.Into("users", "id", "first_name", "last_name"),
				im.Values(sqlite.Arg(1, "Thomas", "Anderson")),
				im.OnConflict("id").DoUpdate(
					im.SetCols("first_name", "last_name").ToQuery(sqlite.Select(
						sm.Columns("first_name", "last_name"),
						sm.From("archived_users"),
						sm.Where(sqlite.Raw("archived_users.id = EXCLUDED.id")),
					)),
				),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "first_name", "last_name") VALUES (?1, ?2, ?3) ON CONFLICT (id) DO UPDATE SET ("first_name", "last_name") = (SELECT first_name, last_name FROM archived_users WHERE archived_users.id = EXCLUDED.id)`,
			ExpectedArgs: []any{1, "Thomas", "Anderson"},
		},
	}

	testutils.RunTests(t, examples, formatter)
}
