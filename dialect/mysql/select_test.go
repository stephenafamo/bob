package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/select/qm"
	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestSelect(t *testing.T) {
	examples := testutils.Testcases{
		"simple select": {
			ExpectedSQL:  "SELECT id, name FROM users WHERE (id IN (?, ?, ?))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mysql.Select(
				qm.Columns("id", "name"),
				qm.From("users"),
				qm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE (id IN (?, ?, ?))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mysql.Select(
				qm.Columns("id", "name"),
				qm.Distinct(),
				qm.From("users"),
				qm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			Query: mysql.Select(
				qm.From(
					mysql.F("generate_series", 1, 3),
				).As("x", "p", "q", "s"),
				qm.OrderBy("p"),
			),
			ExpectedSQL:  `SELECT * FROM generate_series(1, 3) AS ` + "`x`" + ` (` + "`p`" + `, ` + "`q`" + `, ` + "`s`" + `) ORDER BY p`,
			ExpectedArgs: nil,
		},
		"with sub-select": {
			ExpectedSQL: `SELECT status, avg(difference)
					FROM (
						SELECT
							status,
							(LEAD(created_date, 1, NOW())
							OVER (PARTITION BY presale_id ORDER BY created_date)
							 - created_date) AS ` + "`difference`" + `
						FROM presales_presalestatus
					) AS ` + "`differnce_by_status`" + `
					WHERE (status IN ('A', 'B', 'C'))
					GROUP BY status`,
			Query: mysql.Select(
				qm.Columns("status", mysql.F("avg", "difference")),
				qm.From(mysql.Select(
					qm.Columns(
						"status",
						mysql.F("LEAD", "created_date", 1, mysql.F("NOW")).
							Over("").
							PartitionBy("presale_id").
							OrderBy("created_date").
							Minus("created_date").
							As("difference")),
					qm.From("presales_presalestatus")),
				).As("differnce_by_status"),
				qm.Where(mysql.X("status").In(mysql.S("A"), mysql.S("B"), mysql.S("C"))),
				qm.GroupBy("status"),
			),
		},
	}

	testutils.RunTests(t, examples)
}
