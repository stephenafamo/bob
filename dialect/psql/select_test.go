package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/fm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/wm"
	testutils "github.com/stephenafamo/bob/test/utils"
	pgparse "github.com/wasilibs/go-pgquery"
)

var (
	_ bob.Loadable     = &dialect.SelectQuery{}
	_ bob.MapperModder = &dialect.SelectQuery{}
)

func TestSelect(t *testing.T) {
	examples := testutils.Testcases{
		"simple select": {
			Doc:          "Simple Select with some conditions",
			ExpectedSQL:  "SELECT id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(psql.Quote("id").In(psql.Arg(100, 200, 300))),
			),
		},
		"case with else": {
			ExpectedSQL: `SELECT id, name, (CASE WHEN (id = '1') THEN 'A' ELSE 'B' END) AS "C" FROM users`,
			Query: psql.Select(
				sm.Columns(
					"id",
					"name",
					psql.Case().
						When(psql.Quote("id").EQ(psql.S("1")), psql.S("A")).
						Else(psql.S("B")).
						As("C"),
				),
				sm.From("users"),
			),
		},
		"case without else": {
			ExpectedSQL: `SELECT id, name, (CASE WHEN (id = '1') THEN 'A' END) AS "C" FROM users`,
			Query: psql.Select(
				sm.Columns(
					"id",
					"name",
					psql.Case().
						When(psql.Quote("id").EQ(psql.S("1")), psql.S("A")).
						End().
						As("C"),
				),
				sm.From("users"),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.From("users"),
				sm.Where(psql.Quote("id").In(psql.Arg(100, 200, 300))),
			),
		},
		"select distinct on": {
			ExpectedSQL:  "SELECT DISTINCT ON(id) id, name FROM users WHERE (id IN ($1, $2, $3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.Distinct("id"),
				sm.From("users"),
				sm.Where(psql.Quote("id").In(psql.Arg(100, 200, 300))),
			),
		},
		"select from function": {
			Query: psql.Select(
				sm.From(psql.F("generate_series", 1, 3)).As("x", "p", "q", "s"),
			),
			ExpectedSQL:  `SELECT * FROM generate_series(1, 3) AS "x" ("p", "q", "s")`,
			ExpectedArgs: nil,
		},
		"with rows from": {
			Doc: "Select from group of functions. Automatically uses the `ROWS FROM` syntax",
			Query: psql.Select(
				sm.FromFunction(
					psql.F(
						"json_to_recordset",
						psql.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
					)(
						fm.Columns("a", "INTEGER"),
						fm.Columns("b", "TEXT"),
					),
					psql.F("generate_series", 1, 3)(),
				).As("x", "p", "q", "s"),
				sm.OrderBy("p"),
			),
			ExpectedSQL: `SELECT *
				FROM ROWS FROM
					(
						json_to_recordset($1) AS (a INTEGER, b TEXT),
						generate_series(1, 3)
					) AS "x" ("p", "q", "s")
				ORDER BY p`,
			ExpectedArgs: []any{`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`},
		},
		"with sub-select and window": {
			Doc: "Select from subquery with window function",
			ExpectedSQL: `SELECT status, avg(difference)
					FROM (
						SELECT
							status, 
							(LEAD(created_date, 1, NOW())
							OVER(PARTITION BY presale_id ORDER BY created_date)
							 - "created_date") AS "difference"
						FROM presales_presalestatus
					) AS "differnce_by_status"
					WHERE status IN ('A', 'B', 'C')
					GROUP BY status`,
			Query: psql.Select(
				sm.Columns("status", psql.F("avg", "difference")),
				sm.From(psql.Select(
					sm.Columns(
						"status",
						psql.F("LEAD", "created_date", 1, psql.F("NOW"))(
							fm.Over(
								wm.PartitionBy("presale_id"),
								wm.OrderBy("created_date"),
							),
						).Minus(psql.Quote("created_date")).As("difference")),
					sm.From("presales_presalestatus")),
				).As("differnce_by_status"),
				sm.Where(psql.Quote("status").In(psql.S("A"), psql.S("B"), psql.S("C"))),
				sm.GroupBy("status"),
			),
		},
		"select with grouped IN": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(
					psql.Group(psql.Quote("id"), psql.Quote("employee_id")).
						In(psql.ArgGroup(100, 200), psql.ArgGroup(300, 400))),
			),
			ExpectedSQL:  "SELECT id, name FROM users WHERE (id, employee_id) IN (($1, $2), ($3, $4))",
			ExpectedArgs: []any{100, 200, 300, 400},
		},
		"simple limit offset arg": {
			Doc:          "Simple select with limit and offset as argument",
			ExpectedSQL:  "SELECT id, name FROM users LIMIT $1 OFFSET $2",
			ExpectedArgs: []any{10, 15},
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Offset(psql.Arg(15)),
				sm.Limit(psql.Arg(10)),
			),
		},
		"join using": {
			ExpectedSQL: "SELECT id FROM test1 LEFT JOIN test2 USING (id)",
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("test1"),
				sm.LeftJoin("test2").Using("id"),
			),
		},
		"CTE with column aliases": {
			ExpectedSQL: "WITH c(id, data) AS (SELECT id FROM test1 LEFT JOIN test2 USING (id)) SELECT * FROM c",
			Query: psql.Select(
				sm.With("c", "id", "data").As(psql.Select(
					sm.Columns("id"),
					sm.From("test1"),
					sm.LeftJoin("test2").Using("id"),
				)),
				sm.From("c"),
			),
		},
		"Window function over empty frame": {
			ExpectedSQL: "SELECT row_number() OVER () FROM c",
			Query: psql.Select(
				sm.Columns(
					psql.F("row_number")(fm.Over()),
				),
				sm.From("c"),
			),
		},
		"Window function over window name": {
			ExpectedSQL: `SELECT avg(salary) OVER (w)
FROM c 
WINDOW w AS (PARTITION BY depname ORDER BY salary)`,
			Query: psql.Select(
				sm.Columns(
					psql.F("avg", "salary")(fm.Over(wm.BasedOn("w"))),
				),
				sm.From("c"),
				sm.Window("w", wm.PartitionBy("depname"), wm.OrderBy("salary")),
			),
		},
		"select with order by and collate": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.OrderBy("name").Collate("bg-BG-x-icu").Asc(),
			),
			ExpectedSQL: `SELECT id, name FROM users ORDER BY name COLLATE "bg-BG-x-icu" ASC`,
		},
		"with cross join": {
			Query: psql.Select(
				sm.Columns("id", "name", "type"),
				sm.From("users").As("u"),
				sm.CrossJoin(psql.Select(
					sm.Columns("id", "type"),
					sm.From("clients"),
					sm.Where(psql.Quote("client_id").EQ(psql.Arg("123"))),
				)).As("clients"),
				sm.Where(psql.Quote("id").EQ(psql.Arg(100))),
			),
			ExpectedSQL: `SELECT id, name, type
                FROM users AS u CROSS JOIN (
                  SELECT id, type
                  FROM clients
                  WHERE ("client_id" = $1)
                ) AS "clients"
                WHERE ("id" = $2)`,
			ExpectedArgs: []any{"123", 100},
		},
		"with locking": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.ForUpdate("users").SkipLocked(),
			),
			ExpectedSQL: `SELECT id, name FROM users FOR UPDATE OF users SKIP LOCKED`,
		},
		"Multiple Unions": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Union(psql.Select(
					sm.Columns("id", "name"),
					sm.From("admins"),
				)),
				sm.Union(psql.Select(
					sm.Columns("id", "name"),
					sm.From("mods"),
				)),
			),
			ExpectedSQL: `SELECT id, name FROM users UNION select id, name FROM admins UNION select id, name FROM mods`,
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func formatter(s string) (string, error) {
	aTree, err := pgparse.Parse(s)
	if err != nil {
		return "", err
	}

	return pgparse.Deparse(aTree)
}
