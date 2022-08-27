package sqlite_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/insert/qm"
)

func TestInsert(t *testing.T) {
	examples := d.Testcases{
		"simple insert": {
			Query: sqlite.Insert(
				qm.Into("films"),
				qm.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"bulk insert": {
			Query: sqlite.Insert(
				qm.Into("films"),
				qm.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
				qm.Into("films"),
				qm.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.OnConflict(nil).DoNothing(),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (?1, ?2, ?3, ?4, ?5, ?6) ON CONFLICT DO NOTHING",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"upsert": {
			Query: sqlite.Insert(
				qm.IntoAs("distributors", "d", "did", "dname"),
				qm.Values(sqlite.Arg(8, "Anvil Distribution")),
				qm.Values(sqlite.Arg(9, "Sentry Distribution")),
				qm.OnConflict("did").DoUpdate().Set(
					"dname",
					sqlite.Concat(
						"EXCLUDED.dname", sqlite.S(" (formerly "), "d.dname", sqlite.S(")"),
					),
				).Where(sqlite.X("d.zipcode").NE(sqlite.S("21201"))),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES (?1, ?2), (?3, ?4)
				ON CONFLICT (did) DO UPDATE
				SET dname = (EXCLUDED.dname || ' (formerly ' || d.dname || ')')
				WHERE (d.zipcode <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"or replace": {
			Query: sqlite.Insert(
				qm.OrReplace(),
				qm.Into("distributors", "did", "dname"),
				qm.Values(sqlite.Arg(8, "Anvil Distribution")),
				qm.Values(sqlite.Arg(9, "Sentry Distribution")),
			),
			ExpectedSQL: `INSERT OR REPLACE INTO distributors ("did", "dname")
				VALUES (?1, ?2), (?3, ?4)`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
	}

	d.RunTests(t, examples)
}
