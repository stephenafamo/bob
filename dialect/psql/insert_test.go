package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/expr"
)

func TestInsert(t *testing.T) {
	var qm = InsertQM{}
	var examples = d.Testcases{
		"simple insert": {
			Query: Insert(
				qm.Into("films"),
				qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedQuery: "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6)",
			ExpectedArgs:  []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"bulk insert": {
			Query: Insert(
				qm.Into("films"),
				qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedQuery: `INSERT INTO films VALUES
				($1, $2, $3, $4, $5, $6),
				($7, $8, $9, $10, $11, $12)`,
			ExpectedArgs: []any{
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
			},
		},
		"upsert": {
			Query: Insert(
				qm.IntoAs("distributors", "d", "did", "dname"),
				qm.Values(qm.Arg(8, "Anvil Distribution")),
				qm.Values(qm.Arg(9, "Sentry Distribution")),
				qm.OnConflict("did").DoUpdate().Set(
					"dname",
					qm.CONCAT(
						"EXCLUDED.dname", expr.S(" (formerly "), "d.dname", expr.S(")"),
					),
				).Where(qm.X("d.zipcode").NE(expr.S("21201"))),
			),
			ExpectedQuery: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES ($1, $2), ($3, $4)
				ON CONFLICT (did) DO UPDATE
				SET dname = (EXCLUDED.dname || ' (formerly ' || d.dname || ')')
				WHERE (d.zipcode <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"on conflict do nothing": {
			Doc: "Upsert DO NOTHING",
			Query: Insert(
				qm.Into("films"),
				qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.OnConflict(nil).DoNothing(),
			),
			ExpectedQuery: "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING",
			ExpectedArgs:  []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
	}

	d.RunTests(t, examples)
}
