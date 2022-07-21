package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

func TestUpdate(t *testing.T) {
	var qm = UpdateQM{}
	var selectQM = SelectQM{}

	tests := map[string]struct {
		query         query.Query
		expectedQuery string
		expectedArgs  []any
	}{
		"simple": {
			query: Update(
				qm.Table("films"),
				qm.SetArg("kind", "Dramatic"),
				qm.Where(qm.X("kind").EQ(expr.Arg("Drama"))),
			),
			expectedQuery: `UPDATE films SET kind = $1 WHERE (kind = $2)`,
			expectedArgs:  []any{"Dramatic", "Drama"},
		},
		"with from": {
			query: Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.From("accounts"),
				qm.Where(qm.X("accounts.name").EQ(expr.Arg("Acme Corporation"))),
				qm.Where(qm.X("employees.id").EQ("accounts.sales_person")),
			),
			expectedQuery: `UPDATE employees SET sales_count = sales_count + 1 FROM accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			expectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			expectedQuery: `UPDATE employees SET sales_count = sales_count + 1 WHERE (id =
				  (SELECT sales_person FROM accounts WHERE (name = $1)))`,
			expectedArgs: []any{"Acme Corporation"},
			query: Update(
				qm.Table("employees"),
				qm.Set("sales_count", "sales_count + 1"),
				qm.Where(qm.X("id").EQ(expr.P(Select(
					selectQM.Select("sales_person"),
					selectQM.From("accounts"),
					selectQM.Where(qm.X("name").EQ(expr.Arg("Acme Corporation"))),
				)))),
			),
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
