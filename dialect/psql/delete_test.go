package psql

import (
	"testing"

	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

func TestDelete(t *testing.T) {
	var qm = DeleteQM{}

	tests := map[string]struct {
		query         query.Query
		expectedQuery string
		expectedArgs  []any
	}{
		"simple": {
			query: Delete(
				qm.From("films"),
				qm.Where(expr.EQ("kind", expr.Arg("Drama"))),
			),
			expectedQuery: `DELETE FROM films WHERE kind = $1`,
			expectedArgs:  []any{"Drama"},
		},
		"with using": {
			query: Delete(
				qm.From("employees"),
				qm.Using("accounts"),
				qm.Where(expr.EQ("accounts.name", expr.Arg("Acme Corporation"))),
				qm.Where(expr.EQ("employees.id", "accounts.sales_person")),
			),
			expectedQuery: `DELETE FROM employees USING accounts
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
