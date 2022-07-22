package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
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
				qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
			),
			expectedQuery: `DELETE FROM films WHERE (kind = $1)`,
			expectedArgs:  []any{"Drama"},
		},
		"with using": {
			query: Delete(
				qm.From("employees"),
				qm.Using("accounts"),
				qm.Where(qm.X("accounts.name").EQ(qm.Arg("Acme Corporation"))),
				qm.Where(qm.X("employees.id").EQ("accounts.sales_person")),
			),
			expectedQuery: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			expectedArgs: []any{"Acme Corporation"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sql, args, err := query.Build(tc.query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff := d.QueryDiff(tc.expectedQuery, sql); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff := d.ArgsDiff(tc.expectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
