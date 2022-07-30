package mysql_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/mysql"
)

func TestInsert(t *testing.T) {
	var qm = mysql.InsertQM{}

	var examples = d.Testcases{
		"simple insert": {
			Query: mysql.Insert(
				qm.Into("films"),
				qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (?, ?, ?, ?, ?, ?)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"bulk insert": {
			Query: mysql.Insert(
				qm.Into("films"),
				qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL: `INSERT INTO films VALUES
				(?, ?, ?, ?, ?, ?),
				(?, ?, ?, ?, ?, ?)`,
			ExpectedArgs: []any{
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
			},
		},
		"with high priority and ignore modifier": {
			Query: mysql.Insert(
				qm.Into("films"),
				qm.HighPriority(),
				qm.Ignore(),
				qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT HIGH_PRIORITY IGNORE INTO films VALUES (?, ?, ?, ?, ?, ?)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"with optimizer hints": {
			Query: mysql.Insert(
				qm.Into("films"),
				qm.MaxExecutionTime(1000),
				qm.SetVar("cte_max_recursion_depth = 1M"),
				qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL: `INSERT
				/*+
				    MAX_EXECUTION_TIME(1000)
				    SET_VAR(cte_max_recursion_depth = 1M)
				*/ INTO films VALUES (?, ?, ?, ?, ?, ?)`,
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"upsert": {
			Query: mysql.Insert(
				qm.Into("distributors", "did", "dname"),
				qm.Values(mysql.Arg(8, "Anvil Distribution")),
				qm.Values(mysql.Arg(9, "Sentry Distribution")),
				qm.As("new"),
				qm.OnDuplicateKeyUpdate(
					qm.Set("dbname", mysql.Concat(
						"new.dname", mysql.S(" (formerly "), "d.dname", mysql.S(")"),
					)),
				),
			),
			ExpectedSQL: `INSERT INTO distributors (` + "`did`" + `, ` + "`dname`" + `)
				VALUES (?, ?), (?, ?)
				AS new
				ON DUPLICATE KEY UPDATE
				` + "`dbname`" + ` = (new.dname || ' (formerly ' || d.dname || ')')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
	}

	d.RunTests(t, examples)
}
