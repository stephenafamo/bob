package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestSelect(t *testing.T) {
	examples := testutils.Testcases{
		"simple select": {
			ExpectedSQL:  "SELECT id, name FROM users WHERE (id IN (?, ?, ?))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mysql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE (id IN (?, ?, ?))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mysql.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.From("users"),
				sm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			Query: mysql.Select(
				sm.From(
					mysql.F("generate_series", 1, 3),
				).As("x", "p", "q", "s"),
				sm.OrderBy("p"),
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
				sm.Columns("status", mysql.F("avg", "difference")),
				sm.From(mysql.Select(
					sm.Columns(
						"status",
						mysql.F("LEAD", "created_date", 1, mysql.F("NOW")).
							Over("").
							PartitionBy("presale_id").
							OrderBy("created_date").
							Minus("created_date").
							As("difference")),
					sm.From("presales_presalestatus")),
				).As("differnce_by_status"),
				sm.Where(mysql.X("status").In(mysql.S("A"), mysql.S("B"), mysql.S("C"))),
				sm.GroupBy("status"),
			),
		},
	}

	testutils.RunTests(t, examples)
}
