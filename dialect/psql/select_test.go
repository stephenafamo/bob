package psql

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/expr"
)

func TestSelect(t *testing.T) {
	var qm = SelectQM{}
	var examples = d.Testcases{
		"simple select": {
			Doc:           "Simple Select with some conditions",
			ExpectedQuery: "SELECT id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs:  []any{100, 200, 300},
			Query: Select(
				qm.Select("id", "name"),
				qm.From("users"),
				qm.Where(qm.X("id").In(qm.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			Doc: "Select from group of functions. Automatically uses the `ROWS FROM` syntax",
			Query: Select(
				qm.From(
					qm.F(
						"json_to_recordset",
						qm.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
					).Col("a", "INTEGER").Col("b", "TEXT"),
					qm.F("generate_series", 1, 3),
					qm.As("x", "p", "q", "s"),
				),
				qm.OrderBy("p"),
			),
			ExpectedQuery: `SELECT *
				FROM ROWS FROM
					(
						json_to_recordset($1) AS (a INTEGER, b TEXT),
						generate_series(1, 3)
					) AS "x" ("p", "q", "s")
				ORDER BY p`,
			ExpectedArgs: []any{`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`},
		},
		"with sub-select and window": {
			Doc: "Select from subquery with window function",
			ExpectedQuery: `SELECT status, avg(difference)
					FROM (
						SELECT
							status, 
							(LEAD(created_date, 1, NOW())
							OVER(PARTITION BY presale_id ORDER BY created_date)
							 - created_date) AS "difference"
						FROM presales_presalestatus
					) AS "differnce_by_status"
					WHERE (status IN ('A', 'B', 'C'))
					GROUP BY status`,
			Query: Select(
				qm.Select("status", qm.F("avg", "difference")),
				qm.From(Select(
					qm.Select(
						"status",
						qm.F("LEAD", "created_date", 1, qm.F("NOW")).Over(
							expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
						).Minus("created_date").As("difference")),
					qm.From("presales_presalestatus")),
					qm.As("differnce_by_status")),
				qm.Where(qm.X("status").In(qm.S("A"), qm.S("B"), qm.S("C"))),
				qm.GroupBy("status"),
			),
		},
	}

	d.RunTests(t, examples)
}
