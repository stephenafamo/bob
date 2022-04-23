package psql

import (
	"testing"

	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/query"
)

func TestInsert(t *testing.T) {
	var qm = InsertQM{}

	tests := map[string]struct {
		query         query.Query
		expectedQuery string
		expectedArgs  []any
	}{
		"simple insert": {
			query: Insert(
				qm.Into("films"),
				qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			expectedQuery: "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6)",
			expectedArgs:  []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"bulk insert": {
			query: Insert(
				qm.Into("films"),
				qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			expectedQuery: `INSERT INTO films VALUES
				($1, $2, $3, $4, $5, $6),
				($7, $8, $9, $10, $11, $12)`,
			expectedArgs: []any{
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
			},
		},
		"on conflict do nothing": {
			query: Insert(
				qm.Into("films"),
				qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.OnConflict(nil).DoNothing(),
			),
			expectedQuery: "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING",
			expectedArgs:  []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"upsert": {
			query: Insert(
				qm.IntoAs("distributors", "d", "did", "dname"),
				qm.Values(expr.Arg(8, "Anvil Distribution")),
				qm.Values(expr.Arg(9, "Sentry Distribution")),
				qm.OnConflict("did").DoUpdate().SetEQ(
					"dname",
					expr.CONCAT(
						"EXCLUDED.dname", expr.S(" (formerly "), "d.dname", expr.S(")"),
					),
				).Where(expr.NE("d.zipcode", expr.S("21201"))),
			),
			expectedQuery: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES ($1, $2), ($3, $4)
				ON CONFLICT (did) DO UPDATE
				SET dname = EXCLUDED.dname || ' (formerly ' || d.dname || ')'
				WHERE d.zipcode <> '21201'`,
			expectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sql, args, err := query.Build(tc.query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff := queryDiff(tc.expectedQuery, sql); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff := argsDiff(tc.expectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
