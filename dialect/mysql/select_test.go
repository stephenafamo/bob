package mysql_test

import (
	"errors"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/fm"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/wm"
	testutils "github.com/stephenafamo/bob/test/utils"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

var (
	_ bob.Loadable     = &dialect.SelectQuery{}
	_ bob.MapperModder = &dialect.SelectQuery{}
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
		"case with else": {
			ExpectedSQL: "SELECT id, name, (CASE WHEN (`id` = '1') THEN 'A' ELSE 'B' END) AS `C` FROM users",
			Query: mysql.Select(
				sm.Columns(
					"id",
					"name",
					mysql.Case().
						When(mysql.Quote("id").EQ(mysql.S("1")), mysql.S("A")).
						Else(mysql.S("B")).
						As("C"),
				),
				sm.From("users"),
			),
		},
		"case without else": {
			ExpectedSQL: "SELECT id, name, (CASE WHEN (`id` = '1') THEN 'A' END) AS `C` FROM users",
			Query: mysql.Select(
				sm.Columns(
					"id",
					"name",
					mysql.Case().
						When(mysql.Quote("id").EQ(mysql.S("1")), mysql.S("A")).
						End().
						As("C"),
				),
				sm.From("users"),
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
							 - ` + "`created_date`" + `) AS ` + "`difference`" + `
						FROM presales_presalestatus
					` + ") AS `differnce_by_status`" + `
					` + "WHERE (`status` IN ('A', 'B', 'C'))" + `
					GROUP BY status`,
			Query: mysql.Select(
				sm.Columns("status", mysql.F("avg", "difference")),
				sm.From(mysql.Select(
					sm.Columns(
						"status",
						mysql.F("LEAD", "created_date", 1, mysql.F("NOW"))(
							fm.Over(
								wm.PartitionBy("presale_id"),
								wm.OrderBy("created_date"),
							),
						).Minus(mysql.Quote("created_date")).As("difference")),
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
		"select with order by and collate": {
			Query: mysql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.OrderBy("name").Collate("utf8mb4_bg_0900_as_cs").Asc(),
			),
			ExpectedSQL: "SELECT id, name FROM users ORDER BY name COLLATE `utf8mb4_bg_0900_as_cs` ASC",
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func formatter(s string) (string, error) {
	input := antlr.NewInputStream(s)
	lexer := mysqlparser.NewMySqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := mysqlparser.NewMySqlParser(stream)

	el := &errorListener{}
	p.AddErrorListener(el)

	tree := p.Root()
	if el.err != "" {
		return "", errors.New(el.err)
	}

	return tree.GetText(), nil
}

type errorListener struct {
	*antlr.DefaultErrorListener

	err string
}

func (el *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol any, line, column int, msg string, e antlr.RecognitionException) {
	el.err = msg
}
