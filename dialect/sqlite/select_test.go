package sqlite_test

import (
	"errors"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/fm"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/dialect/sqlite/wm"
	testutils "github.com/stephenafamo/bob/test/utils"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

var (
	_ bob.Loadable     = &dialect.SelectQuery{}
	_ bob.MapperModder = &dialect.SelectQuery{}
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
		"case with else": {
			ExpectedSQL: `SELECT id, name, (CASE WHEN ("id" = '1') THEN 'A' ELSE 'B' END) AS "C" FROM users`,
			Query: sqlite.Select(
				sm.Columns(
					"id",
					"name",
					sqlite.Case().
						When(sqlite.Quote("id").EQ(sqlite.S("1")), sqlite.S("A")).
						Else(sqlite.S("B")).
						As("C"),
				),
				sm.From("users"),
			),
		},
		"case without else": {
			ExpectedSQL: `SELECT id, name, (CASE WHEN ("id" = '1') THEN 'A' END) AS "C" FROM users`,
			Query: sqlite.Select(
				sm.Columns(
					"id",
					"name",
					sqlite.Case().
						When(sqlite.Quote("id").EQ(sqlite.S("1")), sqlite.S("A")).
						End().
						As("C"),
				),
				sm.From("users"),
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
						sqlite.F("LEAD", "created_date", 1, sqlite.F("NOW"))(
							fm.Over(
								wm.PartitionBy("presale_id"),
								wm.OrderBy("created_date"),
							),
						).Minus(sqlite.Quote("created_date")).As("difference")),
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
		"select with order by and collate": {
			Query: sqlite.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.OrderBy("name").Collate("NOCASE").Asc(),
			),
			ExpectedSQL: `SELECT id, name FROM users ORDER BY name COLLATE "NOCASE" ASC`,
		},
		"with cross join": {
			Query: sqlite.Select(
				sm.Columns("id", "name", "type"),
				sm.From("users").As("u"),
				sm.CrossJoin(sqlite.Select(
					sm.Columns("id", "type"),
					sm.From("clients"),
					sm.Where(sqlite.Quote("client_id").EQ(sqlite.Arg("123"))),
				)).As("clients"),
				sm.Where(sqlite.Quote("id").EQ(sqlite.Arg(100))),
			),
			ExpectedSQL: `SELECT id, name, type
                FROM users AS "u" CROSS JOIN (
                  SELECT id, type
                  FROM clients
                  WHERE ("client_id" = ?1)
                ) AS "clients"
                WHERE ("id" = ?2)`,
			ExpectedArgs: []any{"123", 100},
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
