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
			ExpectedSQL:  "SELECT id, name FROM users WHERE (`id` IN (?, ?, ?))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mysql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(mysql.Quote("id").In(mysql.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE (`id` IN (?, ?, ?))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mysql.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.From("users"),
				sm.Where(mysql.Quote("id").In(mysql.Arg(100, 200, 300))),
			),
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
					WHERE (` + "`status`" + ` IN ('A', 'B', 'C'))
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
				sm.Where(mysql.Quote("status").In(mysql.S("A"), mysql.S("B"), mysql.S("C"))),
				sm.GroupBy("status"),
			),
		},
		"select with grouped IN": {
			Query: mysql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(mysql.Group(mysql.Quote("id"), mysql.Quote("employee_id")).In(mysql.ArgGroup(100, 200), mysql.ArgGroup(300, 400))),
			),
			ExpectedSQL:  "SELECT id, name FROM users WHERE ((`id`, `employee_id`) IN ((?, ?), (?, ?)))",
			ExpectedArgs: []any{100, 200, 300, 400},
		},
	}

	testutils.RunTests(t, examples, nil)
}

// some bugs to work out for use
//  1. Does not understand multiple tables syntax
//  2. Does not understand aliases in upsert
// In general, TIDB's parser is not updated for MySQL 8.0

// var p = parser.New()

// func formatter(s string) (string, error) {
// node, err := p.ParseOneStmt(s, "", "")
// if err != nil {
// return "", err
// }

// var buf bytes.Buffer
// err = node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf))
// if err != nil {
// return "", err
// }

// return buf.String(), nil
// }
