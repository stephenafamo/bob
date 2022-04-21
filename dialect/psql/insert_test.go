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
		"upsert": {
			query: Insert(
				qm.Into(expr.T("distributors").As("d", "did", "dname")),
				qm.Values(expr.Arg(8), expr.Arg("Anvil Distribution")),
				qm.Values(expr.Arg(9), expr.Arg("Sentry Distribution")),
				qm.OnConflict("did").Do("UPDATE").SetEQ(
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
