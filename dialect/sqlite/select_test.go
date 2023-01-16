package sqlite_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	testutils "github.com/stephenafamo/bob/test_utils"
	_ "modernc.org/sqlite"
)

func TestSelect(t *testing.T) {
	examples := testutils.Testcases{
		"simple select": {
			ExpectedSQL:  "SELECT id, name FROM users WHERE (id IN (?1, ?2, ?3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(sqlite.X("id").In(sqlite.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE (id IN (?1, ?2, ?3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.From("users"),
				sm.Where(sqlite.X("id").In(sqlite.Arg(100, 200, 300))),
			),
		},
		"from function": {
			Query: sqlite.Select(
				sm.From(sqlite.F("generate_series", 1, 3)).As("x"),
			),
			ExpectedSQL:  `SELECT * FROM generate_series(1, 3) AS "x"`,
			ExpectedArgs: nil,
		},
		"with sub-select": {
			ExpectedSQL: `SELECT status, avg(difference)
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
			Query: sqlite.Select(
				sm.Columns("status", sqlite.F("avg", "difference")),
				sm.From(sqlite.Select(
					sm.Columns(
						"status",
						sqlite.F("LEAD", "created_date", 1, sqlite.F("NOW")).
							Over("").
							PartitionBy("presale_id").
							OrderBy("created_date").
							Minus("created_date").
							As("difference")),
					sm.From("presales_presalestatus")),
				).As("differnce_by_status"),
				sm.Where(sqlite.X("status").In(sqlite.S("A"), sqlite.S("B"), sqlite.S("C"))),
				sm.GroupBy("status"),
			),
		},
		"select with grouped IN": {
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(sqlite.Group("id", "employee_id").In(sqlite.ArgGroup(100, 200), sqlite.ArgGroup(300, 400))),
			),
			ExpectedSQL:  "SELECT id, name FROM users WHERE ((id, employee_id) IN ((?1, ?2), (?3, ?4)))",
			ExpectedArgs: []any{100, 200, 300, 400},
		},
	}

	testutils.RunTests(t, examples, formatter)
}

var db, _ = sql.Open("sqlite", ":memory:")

func formatter(s string) (string, error) {
	_, err := db.Exec(s)

	if strings.Contains(err.Error(), "syntax error") {
		return "", err
	}
	return testutils.Clean(s), nil
}
