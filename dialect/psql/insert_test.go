package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestInsert(t *testing.T) {
	examples := testutils.Testcases{
		"simple insert": {
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"insert from select": {
			Query: psql.Insert(
				im.Into("films"),
				im.Query(psql.Select(
					sm.From("tmp_films"),
					sm.Where(psql.Quote("date_prod").LT(psql.Arg("1971-07-13"))),
				)),
			),
			ExpectedSQL:  `INSERT INTO films SELECT * FROM tmp_films WHERE "date_prod" < $1`,
			ExpectedArgs: []any{"1971-07-13"},
		},
		"bulk insert": {
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL: `INSERT INTO films VALUES
				($1, $2, $3, $4, $5, $6),
				($7, $8, $9, $10, $11, $12)`,
			ExpectedArgs: []any{
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
			},
		},
		"upsert": {
			Query: psql.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(psql.Arg(8, "Anvil Distribution")),
				im.Values(psql.Arg(9, "Sentry Distribution")),
				im.OnConflict("did").DoUpdate(
					im.SetCol("dname").To(psql.Concat(
						psql.Raw("EXCLUDED.dname"), psql.S(" (formerly "),
						psql.Quote("d", "dname"), psql.S(")"),
					)),
					im.Where(psql.Quote("d", "zipcode").NE(psql.S("21201"))),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES ($1, $2), ($3, $4)
				ON CONFLICT (did) DO UPDATE
				SET dname = (EXCLUDED.dname || ' (formerly ' || d.dname || ')')
				WHERE (d.zipcode <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"upsert on constraint": {
			Query: psql.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(psql.Arg(8, "Anvil Distribution")),
				im.Values(psql.Arg(9, "Sentry Distribution")),
				im.OnConflictOnConstraint("distributors_pkey").DoUpdate(
					im.SetExcluded("dname"),
					im.Where(psql.Quote("d", "zipcode").NE(psql.S("21201"))),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES ($1, $2), ($3, $4)
				ON CONFLICT ON CONSTRAINT distributors_pkey DO UPDATE
				SET "dname" = EXCLUDED. "dname"
				WHERE (d.zipcode <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"on conflict do nothing": {
			Doc: "Upsert DO NOTHING",
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.OnConflict().DoNothing(),
			),
			ExpectedSQL:  "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
	}

	testutils.RunTests(t, examples, formatter)
}
