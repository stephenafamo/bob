package psql_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/psql"
)

func TestSelect(t *testing.T) {
	qm := psql.SelectQM{}
	examples := d.Testcases{
		"simple select": {
			Doc:          "Simple Select with some conditions",
			ExpectedSQL:  "SELECT id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: psql.Select(
				qm.Columns("id", "name"),
				qm.From("users"),
				qm.Where(psql.X("id").In(psql.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: psql.Select(
				qm.Columns("id", "name"),
				qm.Distinct(),
				qm.From("users"),
				qm.Where(psql.X("id").In(psql.Arg(100, 200, 300))),
			),
		},
		"select distinct on": {
			ExpectedSQL:  "SELECT DISTINCT ON(id) id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: psql.Select(
				qm.Columns("id", "name"),
				qm.Distinct("id"),
				qm.From("users"),
				qm.Where(psql.X("id").In(psql.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			Doc: "Select from group of functions. Automatically uses the `ROWS FROM` syntax",
			Query: psql.Select(
				qm.From(
					psql.F(
						"json_to_recordset",
						psql.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
					).Col("a", "INTEGER").Col("b", "TEXT"),
					psql.F("generate_series", 1, 3),
					qm.As("x", "p", "q", "s"),
				),
				qm.OrderBy("p"),
			),
			ExpectedSQL: `SELECT *
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
			ExpectedSQL: `SELECT status, avg(difference)
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
			Query: psql.Select(
				qm.Columns("status", psql.F("avg", "difference")),
				qm.From(psql.Select(
					qm.Columns(
						"status",
						psql.F("LEAD", "created_date", 1, psql.F("NOW")).
							Over("").
							PartitionBy("presale_id").
							OrderBy("created_date").
							Minus("created_date").
							As("difference")),
					qm.From("presales_presalestatus")),
					qm.As("differnce_by_status")),
				qm.Where(psql.X("status").In(psql.S("A"), psql.S("B"), psql.S("C"))),
				qm.GroupBy("status"),
			),
		},
	}

	d.RunTests(t, examples)
}
