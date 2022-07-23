package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/query"
)

func TestDelete(t *testing.T) {
	var qm = DeleteQM{}
	var examples = d.Testcases{
		"simple": {
			Query: Delete(
				qm.From("films"),
				qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
			),
			ExpectedQuery: `DELETE FROM films WHERE (kind = $1)`,
			ExpectedArgs:  []any{"Drama"},
		},
		"with using": {
			Query: Delete(
				qm.From("employees"),
				qm.Using("accounts"),
				qm.Where(qm.X("accounts.name").EQ(qm.Arg("Acme Corporation"))),
				qm.Where(qm.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedQuery: `DELETE FROM employees USING accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
	}

	for name, tc := range examples {
		t.Run(name, func(t *testing.T) {
			sql, args, err := query.Build(tc.Query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff := d.QueryDiff(tc.ExpectedQuery, sql); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff := d.ArgsDiff(tc.ExpectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
