package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/im"
	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestInsert(t *testing.T) {
	examples := testutils.Testcases{
		"simple insert": {
			Query: mysql.Insert(
				im.Into("films"),
				im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES (?, ?, ?, ?, ?, ?)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"bulk insert": {
			Query: mysql.Insert(
				im.Into("films"),
				im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
				im.Into("films"),
				im.HighPriority(),
				im.Ignore(),
				im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT HIGH_PRIORITY IGNORE INTO films VALUES (?, ?, ?, ?, ?, ?)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"with optimizer hints": {
			Query: mysql.Insert(
				im.Into("films"),
				im.MaxExecutionTime(1000),
				im.SetVar("cte_max_recursion_depth = 1M"),
				im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
				im.Into("distributors", "did", "dname"),
				im.Values(mysql.Arg(8, "Anvil Distribution")),
				im.Values(mysql.Arg(9, "Sentry Distribution")),
				im.As("new"),
				im.OnDuplicateKeyUpdate().
					Set("new", "did").
					SetCol("dbname", mysql.Concat(
						"new.dname", mysql.S(" (formerly "), "d.dname", mysql.S(")"),
					)),
			),
			ExpectedSQL: `INSERT INTO distributors (` + "`did`" + `, ` + "`dname`" + `)
				VALUES (?, ?), (?, ?)
				AS new
				ON DUPLICATE KEY UPDATE
				` + "`did` = `new`.`did`," + `
				` + "`dbname`" + ` = (new.dname || ' (formerly ' || d.dname || ')')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
	}

	// Cannot use the formatter for upsert with alias
	// https://github.com/pingcap/tidb/issues/29259
	testutils.RunTests(t, examples, nil)
}
