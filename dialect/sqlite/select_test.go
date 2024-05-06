package sqlite_test

import (
	"errors"
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func TestSelect(t *testing.T) {
	examples := testutils.Testcases{
		"simple select": {
			ExpectedSQL:  `SELECT id, name FROM users WHERE ("id" IN (?1, ?2, ?3))`,
			ExpectedArgs: []any{100, 200, 300},
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(sqlite.Quote("id").In(sqlite.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  `SELECT DISTINCT id, name FROM users WHERE ("id" IN (?1, ?2, ?3))`,
			ExpectedArgs: []any{100, 200, 300},
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.From("users"),
				sm.Where(sqlite.Quote("id").In(sqlite.Arg(100, 200, 300))),
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
							 - "created_date") AS "difference"
						FROM presales_presalestatus
					) AS "differnce_by_status"
					WHERE ("status" IN ('A', 'B', 'C'))
					GROUP BY status`,
			Query: sqlite.Select(
				sm.Columns("status", sqlite.F("avg", "difference")),
				sm.From(sqlite.Select(
					sm.Columns(
						"status",
						sqlite.F("LEAD", "created_date", 1, sqlite.F("NOW")).
							Over().
							PartitionBy("presale_id").
							OrderBy("created_date").
							Minus(sqlite.Quote("created_date")).
							As("difference")),
					sm.From("presales_presalestatus")),
				).As("differnce_by_status"),
				sm.Where(sqlite.Quote("status").In(sqlite.S("A"), sqlite.S("B"), sqlite.S("C"))),
				sm.GroupBy("status"),
			),
		},
		"select with grouped IN": {
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(sqlite.Group(sqlite.Quote("id"), sqlite.Quote("employee_id")).In(sqlite.ArgGroup(100, 200), sqlite.ArgGroup(300, 400))),
			),
			ExpectedSQL:  `SELECT id, name FROM users WHERE (("id", "employee_id") IN ((?1, ?2), (?3, ?4)))`,
			ExpectedArgs: []any{100, 200, 300, 400},
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func formatter(s string) (string, error) {
	input := antlr.NewInputStream(s)
	lexer := sqliteparser.NewSQLiteLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := sqliteparser.NewSQLiteParser(stream)

	el := &errorListener{}
	p.AddErrorListener(el)

	tree := p.Parse()
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
