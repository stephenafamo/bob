package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/im"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
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
		"insert from select": {
			Query: mysql.Insert(
				im.Into("films"),
				im.Query(mysql.Select(
					sm.From("tmp_films"),
					sm.Where(mysql.Quote("date_prod").LT(mysql.Arg("1971-07-13"))),
				)),
			),
			ExpectedSQL:  "INSERT INTO films SELECT * FROM tmp_films WHERE (`date_prod` < ?)",
			ExpectedArgs: []any{"1971-07-13"},
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
				im.OnDuplicateKeyUpdate(
					im.UpdateWithAlias("new", "did"),
					im.UpdateCol("dbname").To(mysql.Concat(
						mysql.Quote("new", "dname"), mysql.S(" (formerly "),
						mysql.Quote("d", "dname"), mysql.S(")"),
					)),
				),
			),
			ExpectedSQL: `INSERT INTO distributors (` + "`did`" + `, ` + "`dname`" + `)
				VALUES (?, ?), (?, ?)
				AS ` + "`new`" + `
				ON DUPLICATE KEY UPDATE
				` + "`did` = `new`.`did`," + `
				` + "`dbname` = (`new`.`dname` || ' (formerly ' || `d`.`dname` || ')')",
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"insert with quoted row alias": {
			Query: mysql.Insert(
				im.Into("t", "c"),
				im.Values(mysql.Arg(1)),
				im.As("row alias"),
			),
			ExpectedSQL:  "INSERT INTO t(`c`) VALUES (?) AS `row alias`",
			ExpectedArgs: []any{1},
		},
		"insert with partition": {
			Query: mysql.Insert(
				im.Into("films"),
				im.Partition("part one"),
				im.Values(mysql.Arg(1)),
			),
			ExpectedSQL:  "INSERT INTO films PARTITION (`part one`) VALUES (?)",
			ExpectedArgs: []any{1},
		},
		"upsert2": {
			Query: mysql.Insert(
				im.Into("distributors", "did", "dname"),
				im.Values(mysql.Arg(8, "Anvil Distribution")),
				im.Values(mysql.Arg(9, "Sentry Distribution")),
				im.OnDuplicateKeyUpdate(im.UpdateWithValues("did", "dbname")),
			),
			ExpectedSQL: `INSERT INTO distributors (` + "`did`" + `, ` + "`dname`" + `)
				VALUES (?, ?), (?, ?)
				ON DUPLICATE KEY UPDATE
				` + "`did` = VALUES(`did`)," + `
				` + "`dbname` = VALUES(`dbname`)",
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"on duplicate key update column with spaces": {
			Doc: "UpdateCol quotes identifiers so column names may contain spaces",
			Query: mysql.Insert(
				im.Into("items", "id"),
				im.Values(mysql.Arg(1, "first")),
				im.OnDuplicateKeyUpdate(
					im.UpdateCol("display name").ToArg("renamed"),
				),
			),
			ExpectedSQL: `INSERT INTO items (` + "`id`" + `) VALUES (?, ?)
				ON DUPLICATE KEY UPDATE
				` + "`display name` = ?",
			ExpectedArgs: []any{1, "first", "renamed"},
		},
		"on duplicate key updateCol as mod": {
			Query: mysql.Insert(
				im.Into("distributors", "did", "dname"),
				im.Values(mysql.Arg(8, "Anvil Distribution")),
				im.OnDuplicateKeyUpdate(
					im.UpdateCol("dname").ToArg("updated"),
				),
			),
			ExpectedSQL: `INSERT INTO distributors (` + "`did`" + `, ` + "`dname`" + `)
				VALUES (?, ?)
				ON DUPLICATE KEY UPDATE
				` + "`dname` = ?",
			ExpectedArgs: []any{8, "Anvil Distribution", "updated"},
		},
		"on duplicate key updateCol via Update helper": {
			Query: mysql.Insert(
				im.Into("distributors", "did", "dname"),
				im.Values(mysql.Arg(8, "Anvil Distribution")),
				im.OnDuplicateKeyUpdate(
					im.Update(
						im.UpdateCol("dname").ToArg("updated"),
						im.UpdateCol("did").ToArg(8),
					),
				),
			),
			ExpectedSQL: `INSERT INTO distributors (` + "`did`" + `, ` + "`dname`" + `)
				VALUES (?, ?)
				ON DUPLICATE KEY UPDATE
				` + "`dname` = ?," + `
				` + "`did` = ?",
			ExpectedArgs: []any{8, "Anvil Distribution", "updated", 8},
		},
	}

	// Cannot use the formatter for upsert with alias
	// https://github.com/pingcap/tidb/issues/29259
	testutils.RunTests(t, examples, formatter)
}
