package sqlite

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/expr"
)

func TestSelect(t *testing.T) {
	var qm = SelectQM{}

	var examples = d.Testcases{
		"simple select": {
			ExpectedQuery: "SELECT id, name FROM users WHERE (id IN (?1, ?2, ?3))",
			ExpectedArgs:  []any{100, 200, 300},
			Query: Select(
				qm.Select("id", "name"),
				qm.From("users"),
				qm.Where(qm.X("id").In(qm.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			Query: Select(
				qm.From(
					qm.F("generate_series", 1, 3),
					qm.As("x", "p", "q", "s"),
				),
				qm.OrderBy("p"),
			),
			ExpectedQuery: `SELECT * FROM generate_series(1, 3) AS "x" ("p", "q", "s") ORDER BY p`,
			ExpectedArgs:  nil,
		},
		"with sub-select": {
			ExpectedQuery: `SELECT status, avg(difference)
					FROM (
						SELECT
						status,
						(LEAD(created_date, 1, NOW())
						OVER (PARTITION BY presale_id ORDER BY created_date)
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
					qm.As("differnce_by_status"),
				),
				qm.Where(qm.X("status").In(qm.S("A"), qm.S("B"), qm.S("C"))),
				qm.GroupBy("status"),
			),
		},
	}

	d.RunTests(t, examples)
}
