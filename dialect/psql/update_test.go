package psql

import (
	"testing"

	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/query"
)

func TestUpdate(t *testing.T) {
	var qm = UpdateQM{}

	tests := map[string]struct {
		query         query.Query
		expectedQuery string
		expectedArgs  []any
	}{
		"simple": {
			query: Update(
				qm.Table("films"),
				qm.SetEQ("kind", expr.Arg("Dramatic")),
				qm.Where(expr.EQ("kind", expr.Arg("Drama"))),
			),
			expectedQuery: `UPDATE films SET kind = $1 WHERE kind = $2`,
			expectedArgs:  []any{"Dramatic", "Drama"},
		},
		"with from": {
			query: Update(
				qm.Table("employees"),
				qm.SetEQ("sales_count", "sales_count + 1"),
				qm.From("accounts"),
				qm.Where(expr.EQ("accounts.name", expr.Arg("Acme Corporation"))),
				qm.Where(expr.EQ("employees.id", "accounts.sales_person")),
			),
			expectedQuery: `UPDATE employees SET sales_count = sales_count + 1 FROM accounts
			  WHERE accounts.name = $1
			  AND employees.id = accounts.sales_person`,
			expectedArgs: []any{"Acme Corporation"},
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
