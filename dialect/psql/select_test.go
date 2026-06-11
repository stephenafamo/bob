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
		"from replaces previous alias": {
			Doc:         "A later From without alias replaces the whole table ref",
			ExpectedSQL: "SELECT id FROM orders",
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users").As("u"),
				sm.From("orders"),
			),
		},
		"from keeps existing joins": {
			Doc:         "A later From replaces the primary table but keeps joins already on the from_item",
			ExpectedSQL: "SELECT id FROM orders CROSS JOIN events",
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users"),
				sm.CrossJoin("events"),
				sm.From("orders"),
			),
		},
		"from multiple tables cross join": {
			Doc:         "Multiple tables in FROM via CROSS JOIN on the primary from_item",
			ExpectedSQL: "SELECT id FROM users CROSS JOIN orders",
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users"),
				sm.CrossJoin("orders"),
			),
		},
		"from multiple tables cross join unordered calls": {
			Doc:         "Multiple tables in FROM via CROSS JOIN on the primary from_item",
			ExpectedSQL: "SELECT id FROM users CROSS JOIN orders",
			Query: psql.Select(
				sm.Columns("id"),
				sm.CrossJoin("orders"),
				sm.From("users"),
			),
		},
		"from function": {
			Doc:         "FROM a single table function via TableFunctions",
			ExpectedSQL: "SELECT p FROM generate_series(1, 3)",
			Query: psql.Select(
				sm.Columns("p"),
				sm.From(sm.FromFunction(psql.F("generate_series", 1, 3)())),
			),
		},
		"function qualified name": {
			Query: psql.Select(
				sm.Columns(psql.F(psql.Quote("pg_catalog", "array_agg"), "x")()),
			),
			ExpectedSQL: `SELECT "pg_catalog"."array_agg"("x")`,
		},
		"from then standalone inner join": {
			Doc:         "FROM with INNER JOIN applied as a standalone mod after From",
			ExpectedSQL: `SELECT id FROM users INNER JOIN events ON ("users"."id" = "events"."user_id")`,
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users"),
				sm.InnerJoin("events").OnEQ(
					psql.Quote("users", "id"),
					psql.Quote("events", "user_id"),
				),
			),
		},
		"from with inline inner join": {
			Doc:         "FROM with INNER JOIN attached inline",
			ExpectedSQL: `SELECT id FROM users INNER JOIN events ON ("users"."id" = "events"."user_id")`,
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users", sm.InnerJoin("events").OnEQ(
					psql.Quote("users", "id"),
					psql.Quote("events", "user_id"),
				)),
			),
		},
		"from inline joins keep existing standalone joins": {
			Doc:         "Replacing FROM with inline joins preserves already attached standalone joins",
			ExpectedSQL: `SELECT id FROM users CROSS JOIN events INNER JOIN admins ON ("users"."id" = "admins"."user_id")`,
			Query: psql.Select(
				sm.Columns("id"),
				sm.CrossJoin("events"),
				sm.From("users", sm.InnerJoin("admins").OnEQ(
					psql.Quote("users", "id"),
					psql.Quote("admins", "user_id"),
				)),
			),
		},
		"standalone inner join before from": {
			Doc:         "INNER JOIN applied as a standalone mod before From",
			ExpectedSQL: `SELECT id FROM users INNER JOIN events ON ("users"."id" = "events"."user_id")`,
			Query: psql.Select(
				sm.Columns("id"),
				sm.InnerJoin("events").OnEQ(
					psql.Quote("users", "id"),
					psql.Quote("events", "user_id"),
				),
				sm.From("users"),
			),
		},
		"from rows from with cross join": {
			Doc: "FROM ROWS FROM (...) with an additional table via CROSS JOIN",
			Query: psql.Select(
				sm.Columns("id"),
				sm.From(sm.FromFunction(
					psql.F("generate_series", 1, 1)(),
					psql.F("json_to_recordset", psql.Arg(`[{"a":1}]`))(fm.Columns("a", "INTEGER")),
				)),
				sm.CrossJoin("orders"),
			),
			ExpectedSQL:  `SELECT id FROM ROWS FROM (generate_series(1, 1), json_to_recordset($1) AS (a INTEGER)) CROSS JOIN orders`,
			ExpectedArgs: []any{`[{"a":1}]`},
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
		"select from tablesample repeatable": {
			Query: psql.Select(
				sm.From("users").TableSample("BERNOULLI", psql.Arg(50)).Repeatable(psql.Arg(7)),
			),
			ExpectedSQL:  `SELECT * FROM users TABLESAMPLE BERNOULLI($1) REPEATABLE ($2)`,
			ExpectedArgs: []any{50, 7},
		},
		"select from tablesample repeatable helper": {
			Query: psql.Select(
				sm.From("users").TableSampleBernoulli(psql.Arg(50)).Repeatable(psql.Arg(7)),
			),
			ExpectedSQL:  `SELECT * FROM users TABLESAMPLE BERNOULLI($1) REPEATABLE ($2)`,
			ExpectedArgs: []any{50, 7},
		},
		"select join with tablesample": {
			Query: psql.Select(
				sm.From("users"),
				sm.InnerJoin("events").TableSample("SYSTEM", 10).On(
					psql.Quote("users", "id").EQ(psql.Quote("events", "user_id")),
				),
			),
			ExpectedSQL: `SELECT * FROM users INNER JOIN events TABLESAMPLE SYSTEM(10) ON ("users"."id" = "events"."user_id")`,
		},
		"select join with tablesample helper": {
			Query: psql.Select(
				sm.From("users"),
				sm.InnerJoin("events").TableSampleSystem(10).On(
					psql.Quote("users", "id").EQ(psql.Quote("events", "user_id")),
				),
			),
			ExpectedSQL: `SELECT * FROM users INNER JOIN events TABLESAMPLE SYSTEM(10) ON ("users"."id" = "events"."user_id")`,
		},
		"with rows from": {
			Doc: "Select from group of functions. Automatically uses the `ROWS FROM` syntax",
			Query: psql.Select(
				sm.From(sm.FromFunction(
					psql.F(
						"json_to_recordset",
						psql.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
					)(
						fm.Columns("a", "INTEGER"),
						fm.Columns("b", "TEXT"),
					),
					psql.F("generate_series", 1, 3)(),
				)).As("x", "p", "q", "s"),
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
					WHERE ("status" IN ('A', 'B', 'C'))
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
		"select with aliased subquery in columns": {
			ExpectedSQL: `SELECT COUNT(*) AS "all", (SELECT COUNT(*) AS "c" FROM teams WHERE (active = $1)) AS "active_count" FROM teams`,
			Query: psql.Select(
				sm.Columns(
					psql.Raw("COUNT(*)").As("all"),
					psql.Select(
						sm.Columns(psql.Raw("COUNT(*)").As("c")),
						sm.From("teams"),
						sm.Where(psql.Raw("active").EQ(psql.Arg(true))),
					).As("active_count"),
				),
				sm.From("teams"),
			),
			ExpectedArgs: []any{true},
		},
		"select with grouped IN": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(
					psql.Group(psql.Quote("id"), psql.Quote("employee_id")).
						In(psql.ArgGroup(100, 200), psql.ArgGroup(300, 400))),
			),
			ExpectedSQL:  `SELECT id, name FROM users WHERE (("id", "employee_id") IN (($1, $2), ($3, $4)))`,
			ExpectedArgs: []any{100, 200, 300, 400},
		},
		"group by grouped expression list": {
			Query: psql.Select(
				sm.Columns("region", psql.Raw("COUNT(*)")),
				sm.From("sales"),
				sm.GroupBy(sm.Grouping("region", "product")),
			),
			ExpectedSQL: `SELECT region, COUNT(*) FROM sales GROUP BY (region, product)`,
		},
		"group by rollup with nested group": {
			Query: psql.Select(
				sm.Columns("region", "product", psql.Raw("COUNT(*)")),
				sm.From("sales"),
				sm.GroupBy(sm.Rollup("region", sm.Grouping("product", "segment"))),
			),
			ExpectedSQL: `SELECT region, product, COUNT(*) FROM sales GROUP BY ROLLUP (region, (product, segment))`,
		},
		"group by grouping sets": {
			Query: psql.Select(
				sm.Columns("region", "product", psql.Raw("COUNT(*)")),
				sm.From("sales"),
				sm.GroupBy(sm.GroupingSets("region", sm.Grouping("product", "segment"), "()")),
			),
			ExpectedSQL: `SELECT region, product, COUNT(*) FROM sales GROUP BY GROUPING SETS (region, (product, segment), ())`,
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
		"join using alias": {
			ExpectedSQL: `SELECT id FROM test1 LEFT JOIN test2 USING (id) AS "joined_id"`,
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("test1"),
				sm.LeftJoin("test2").UsingAs("joined_id", "id"),
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
		"CTE with search breadth": {
			ExpectedSQL: "WITH c(id) AS (SELECT id FROM test1) SEARCH BREADTH FIRST BY id SET order_col SELECT * FROM c",
			Query: psql.Select(
				sm.With("c", "id").As(psql.Select(
					sm.Columns("id"),
					sm.From("test1"),
				)).SearchBreadth("order_col", "id"),
				sm.From("c"),
			),
		},
		"CTE with search depth": {
			ExpectedSQL: "WITH c(id) AS (SELECT id FROM test1) SEARCH DEPTH FIRST BY id SET order_col SELECT * FROM c",
			Query: psql.Select(
				sm.With("c", "id").As(psql.Select(
					sm.Columns("id"),
					sm.From("test1"),
				)).SearchDepth("order_col", "id"),
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
		"CTE with quoted name and columns": {
			Query: psql.Select(
				sm.With("my cte", "a col").As(psql.Select(
					sm.Columns(psql.Quote("a col")),
					sm.From("t"),
				)),
				sm.From(psql.Quote("my cte")),
			),
			ExpectedSQL: `WITH "my cte"("a col") AS (SELECT "a col" FROM t) SELECT * FROM "my cte"`,
		},
		"named window with quoted name": {
			Query: psql.Select(
				sm.Columns("x"),
				sm.From("t"),
				sm.Window("my win", wm.OrderBy("x")),
			),
			ExpectedSQL: `SELECT x FROM t WINDOW "my win" AS (ORDER BY x)`,
		},
		"FOR UPDATE OF quoted table": {
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users"),
				sm.ForUpdate(psql.Quote("my table")),
			),
			ExpectedSQL: `SELECT id FROM users FOR UPDATE OF "my table"`,
		},
		"FOR UPDATE OF qualified table": {
			Query: psql.Select(
				sm.Columns("id"),
				sm.From("users"),
				sm.ForUpdate(psql.Quote("public", "users")),
			),
			ExpectedSQL: `SELECT id FROM users FOR UPDATE OF "public"."users"`,
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
			ExpectedSQL: `SELECT id, name FROM users UNION (SELECT id, name FROM admins) UNION (SELECT id, name FROM mods)`,
		},
		"Union with combined args": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Limit(100),
				sm.OrderBy("id"),
				sm.Union(psql.Select(
					sm.Columns("id", "name"),
					sm.From("admins"),
					sm.Limit(10),
					sm.OrderBy("id"),
				)),
				sm.OrderCombined("id"),
				sm.LimitCombined(1000),
			),
			ExpectedSQL: `(SELECT id, name FROM users ORDER BY id LIMIT 100) UNION (SELECT id, name FROM admins ORDER BY id LIMIT 10)
ORDER BY id LIMIT 1000`,
		},
		"Union with uncombined args": {
			Query: psql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Limit(1),
				sm.OrderBy("id"),
				sm.Union(psql.Select(
					sm.Columns("id", "name"),
					sm.From("admins"),
					sm.Limit(1),
					sm.OrderBy("id"),
				)),
			),
			ExpectedSQL: `(SELECT id, name FROM users ORDER BY id LIMIT 1) UNION (SELECT id, name FROM admins ORDER BY id LIMIT 1)`,
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func formatter(s string) (string, error) {
	aTree, err := pgparse.Parse(s)
	if err == nil {
		return pgparse.Deparse(aTree)
	}
	// Parser may not support newer syntax (e.g. RETURNING WITH); fall back to Clean.
	return testutils.Clean(s), nil
}
