package mssql_test

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mssql"
	"github.com/stephenafamo/bob/dialect/mssql/fm"
	"github.com/stephenafamo/bob/dialect/mssql/sm"
	"github.com/stephenafamo/bob/dialect/mssql/wm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestSelect(t *testing.T) {
	examples := testutils.Testcases{
		"simple select": {
			Doc:          "Simple Select with some conditions",
			ExpectedSQL:  "SELECT id, name FROM users WHERE ([id] IN (@p1, @p2, @p3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(mssql.Quote("id").In(mssql.Arg(100, 200, 300))),
			),
		},
		"select distinct": {
			ExpectedSQL:  "SELECT DISTINCT id, name FROM users WHERE ([id] IN (@p1, @p2, @p3))",
			ExpectedArgs: []any{100, 200, 300},
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.From("users"),
				sm.Where(mssql.Quote("id").In(mssql.Arg(100, 200, 300))),
			),
		},
		"select top": {
			Doc:         "Select with TOP clause",
			ExpectedSQL: "SELECT TOP (10) id, name FROM users",
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.Top(10),
				sm.From("users"),
			),
		},
		"select distinct top": {
			Doc:         "DISTINCT appears before TOP in SELECT clause",
			ExpectedSQL: "SELECT DISTINCT TOP (5) id, name FROM users",
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.Distinct(),
				sm.Top(5),
				sm.From("users"),
			),
		},
		"select top percent": {
			Doc:         "Select with TOP PERCENT clause",
			ExpectedSQL: "SELECT TOP (50) PERCENT id, name FROM users",
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.TopPercent(50),
				sm.From("users"),
			),
		},
		"select top with ties": {
			Doc:         "Select with TOP WITH TIES (requires ORDER BY)",
			ExpectedSQL: "SELECT TOP (5) WITH TIES id, salary FROM employees ORDER BY salary DESC",
			Query: mssql.Select(
				sm.Columns("id", "salary"),
				sm.TopWithTies(5),
				sm.From("employees"),
				sm.OrderBy("salary").Desc(),
			),
		},
		"offset fetch": {
			Doc:          "Select with OFFSET/FETCH NEXT pagination",
			ExpectedSQL:  "SELECT id, name FROM users ORDER BY id OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY",
			ExpectedArgs: []any{10, 5},
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.OrderBy("id"),
				sm.Offset(mssql.Arg(10)),
				sm.Fetch(mssql.Arg(5)),
			),
		},
		"top with offset fetch": {
			Doc:          "TOP and OFFSET/FETCH can coexist — TOP limits before ORDER BY, OFFSET/FETCH paginate after",
			ExpectedSQL:  "SELECT TOP (100) id, name FROM users ORDER BY id OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY",
			ExpectedArgs: []any{20, 10},
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.Top(100),
				sm.From("users"),
				sm.OrderBy("id"),
				sm.Offset(mssql.Arg(20)),
				sm.Fetch(mssql.Arg(10)),
			),
		},
		"case with else": {
			ExpectedSQL: `SELECT id, name, (CASE WHEN ([id] = 'A') THEN 'Yes' ELSE 'No' END) AS [C] FROM users`,
			Query: mssql.Select(
				sm.Columns(
					"id",
					"name",
					mssql.Case().
						When(mssql.Quote("id").EQ(mssql.S("A")), mssql.S("Yes")).
						Else(mssql.S("No")).
						As("C"),
				),
				sm.From("users"),
			),
		},
		"join using on": {
			ExpectedSQL: "SELECT id FROM test1 INNER JOIN test2 ON ([test1].[id] = [test2].[id])",
			Query: mssql.Select(
				sm.Columns("id"),
				sm.From("test1"),
				sm.InnerJoin("test2").OnEQ(mssql.Quote("test1", "id"), mssql.Quote("test2", "id")),
			),
		},
		"left join": {
			ExpectedSQL:  `SELECT u.id, o.total FROM users AS [u] LEFT JOIN orders AS [o] ON ([u].[id] = [o].[user_id]) WHERE ([o].[total] > @p1)`,
			ExpectedArgs: []any{100},
			Query: mssql.Select(
				sm.Columns("u.id", "o.total"),
				sm.From("users").As("u"),
				sm.LeftJoin("orders").As("o").OnEQ(mssql.Quote("u", "id"), mssql.Quote("o", "user_id")),
				sm.Where(mssql.Quote("o", "total").GT(mssql.Arg(100))),
			),
		},
		"CTE with column aliases": {
			ExpectedSQL: "WITH [c]([id], [data]) AS (SELECT id FROM test1 INNER JOIN test2 ON ([test1].[id] = [test2].[id])) SELECT * FROM c",
			Query: mssql.Select(
				sm.With("c", "id", "data").As(mssql.Select(
					sm.Columns("id"),
					sm.From("test1"),
					sm.InnerJoin("test2").OnEQ(mssql.Quote("test1", "id"), mssql.Quote("test2", "id")),
				)),
				sm.From("c"),
			),
		},
		"union": {
			ExpectedSQL:  "SELECT id, name FROM users WHERE ([status] = @p1) UNION (SELECT id, name FROM archived_users WHERE ([status] = @p2))",
			ExpectedArgs: []any{"active", "archived"},
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(mssql.Quote("status").EQ(mssql.Arg("active"))),
				sm.Union(mssql.Select(
					sm.Columns("id", "name"),
					sm.From("archived_users"),
					sm.Where(mssql.Quote("status").EQ(mssql.Arg("archived"))),
				)),
			),
		},
		"union all": {
			ExpectedSQL: "SELECT 1 UNION ALL (SELECT 2)",
			Query: mssql.Select(
				sm.Columns(1),
				sm.UnionAll(mssql.Select(
					sm.Columns(2),
				)),
			),
		},
		"Window function over empty frame": {
			ExpectedSQL: "SELECT row_number() OVER () FROM c",
			Query: mssql.Select(
				sm.Columns(
					mssql.F("row_number")(fm.Over()),
				),
				sm.From("c"),
			),
		},
		"Window function with partition": {
			ExpectedSQL: `SELECT avg(salary) OVER (PARTITION BY depname ORDER BY salary) FROM employees`,
			Query: mssql.Select(
				sm.Columns(
					mssql.F("avg", "salary")(fm.Over(
						wm.PartitionBy("depname"),
						wm.OrderBy("salary"),
					)),
				),
				sm.From("employees"),
			),
		},
		"select with grouped IN": {
			Query: mssql.Select(
				sm.Columns("id", "name"),
				sm.From("users"),
				sm.Where(
					mssql.Group(mssql.Quote("id"), mssql.Quote("employee_id")).
						In(mssql.ArgGroup(100, 200), mssql.ArgGroup(300, 400)),
				),
			),
			ExpectedSQL:  "SELECT id, name FROM users WHERE (([id], [employee_id]) IN ((@p1, @p2), (@p3, @p4)))",
			ExpectedArgs: []any{100, 200, 300, 400},
		},
		"subquery in from": {
			ExpectedSQL:  `SELECT s.id FROM (SELECT id FROM users WHERE ([active] = @p1)) AS [s]`,
			ExpectedArgs: []any{true},
			Query: mssql.Select(
				sm.Columns("s.id"),
				sm.From(mssql.Select(
					sm.Columns("id"),
					sm.From("users"),
					sm.Where(mssql.Quote("active").EQ(mssql.Arg(true))),
				)).As("s"),
			),
		},
		"string concat with plus": {
			Doc:         "MSSQL uses + for string concatenation",
			ExpectedSQL: `SELECT ([first_name] + ' ' + [last_name]) AS [full_name] FROM users`,
			Query: mssql.Select(
				sm.Columns(mssql.Concat(
					mssql.Quote("first_name"),
					mssql.S(" "),
					mssql.Quote("last_name"),
				).As("full_name")),
				sm.From("users"),
			),
		},
		"having clause": {
			ExpectedSQL:  "SELECT department, COUNT(*) FROM employees GROUP BY department HAVING COUNT(*) > @p1",
			ExpectedArgs: []any{5},
			Query: mssql.Select(
				sm.Columns("department", "COUNT(*)"),
				sm.From("employees"),
				sm.GroupBy("department"),
				sm.Having(mssql.Raw("COUNT(*) > ?", 5)),
			),
		},
		"cross join": {
			ExpectedSQL: "SELECT * FROM t1 CROSS JOIN t2",
			Query: mssql.Select(
				sm.From("t1"),
				sm.CrossJoin("t2"),
			),
		},
		"intersect": {
			ExpectedSQL: "SELECT id FROM t1 INTERSECT (SELECT id FROM t2)",
			Query: mssql.Select(
				sm.Columns("id"),
				sm.From("t1"),
				sm.Intersect(mssql.Select(
					sm.Columns("id"),
					sm.From("t2"),
				)),
			),
		},
		"except": {
			ExpectedSQL: "SELECT id FROM t1 EXCEPT (SELECT id FROM t2)",
			Query: mssql.Select(
				sm.Columns("id"),
				sm.From("t1"),
				sm.Except(mssql.Select(
					sm.Columns("id"),
					sm.From("t2"),
				)),
			),
		},
		"not expression": {
			ExpectedSQL: "SELECT NOT true",
			Query: mssql.Select(
				sm.Columns(mssql.Not(mssql.Raw("true"))),
			),
		},
		"or expression": {
			ExpectedSQL: "SELECT * FROM users WHERE (a OR b)",
			Query: mssql.Select(
				sm.From("users"),
				sm.Where(mssql.Or(mssql.Raw("a"), mssql.Raw("b"))),
			),
		},
		"and expression": {
			ExpectedSQL: "SELECT * FROM users WHERE (a AND b)",
			Query: mssql.Select(
				sm.From("users"),
				sm.Where(mssql.And(mssql.Raw("a"), mssql.Raw("b"))),
			),
		},
		"cast expression": {
			ExpectedSQL: "SELECT (CAST([price] AS DECIMAL(10,2))) FROM products",
			Query: mssql.Select(
				sm.Columns(mssql.Cast(mssql.Quote("price"), "DECIMAL(10,2)")),
				sm.From("products"),
			),
		},
		"exists expression": {
			ExpectedSQL:  "SELECT * FROM users WHERE EXISTS ((SELECT 1 FROM orders WHERE ([user_id] = @p1)))",
			ExpectedArgs: []any{42},
			Query: mssql.Select(
				sm.From("users"),
				sm.Where(mssql.Exists(mssql.Select(
					sm.Columns(1),
					sm.From("orders"),
					sm.Where(mssql.Quote("user_id").EQ(mssql.Arg(42))),
				))),
			),
		},
		"minus expression": {
			ExpectedSQL:  "SELECT - @p1",
			ExpectedArgs: []any{1},
			Query: mssql.Select(
				sm.Columns(mssql.Minus(mssql.Arg(1))),
			),
		},
		"placeholder": {
			ExpectedSQL:  "SELECT @p1, @p2, @p3",
			ExpectedArgs: []any{nil, nil, nil},
			Query: mssql.Select(
				sm.Columns(mssql.Placeholder(3)),
			),
		},
		"raw query": {
			ExpectedSQL:  "SELECT * FROM users WHERE id = @p1",
			ExpectedArgs: []any{42},
			Query:        mssql.RawQuery("SELECT * FROM users WHERE id = ?", 42),
		},
	}

	testutils.RunTests(t, examples, nil)
}

// TestNamedArgInQuery exercises @name placeholders embedded in a constructed
// SELECT query alongside positional placeholders. The bound query writer must
// route sql.NamedArg values through Dialect.WriteNamedArg, mixing them with
// regular @p-prefixed positional args without renumbering the named ones.
func TestNamedArgInQuery(t *testing.T) {
	q := mssql.Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Where(mssql.And(
			mssql.Quote("id").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
				return bob.Express(ctx, w, d, start, sql.Named("user_id", 42))
			})),
			mssql.Quote("status").EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
				return bob.Express(ctx, w, d, start, sql.Named("status_name", "active"))
			})),
		)),
		sm.Where(mssql.Quote("role").EQ(mssql.Arg("admin"))),
	)

	gotSQL, gotArgs, err := q.Build(context.Background())
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	got := strings.Join(strings.Fields(gotSQL), " ")
	if !strings.Contains(got, "@user_id") {
		t.Fatalf("expected @user_id named placeholder in SQL, got: %s", got)
	}
	if !strings.Contains(got, "@status_name") {
		t.Fatalf("expected @status_name named placeholder in SQL, got: %s", got)
	}
	if !strings.Contains(got, "@p") {
		t.Fatalf("expected @p positional placeholder for role arg in SQL, got: %s", got)
	}

	if len(gotArgs) != 3 {
		t.Fatalf("expected 3 args, got %d: %#v", len(gotArgs), gotArgs)
	}

	n0, ok := gotArgs[0].(sql.NamedArg)
	if !ok || n0.Name != "user_id" || n0.Value != 42 {
		t.Errorf("arg[0]: expected sql.Named(user_id, 42), got %#v", gotArgs[0])
	}
	n1, ok := gotArgs[1].(sql.NamedArg)
	if !ok || n1.Name != "status_name" || n1.Value != "active" {
		t.Errorf("arg[1]: expected sql.Named(status_name, active), got %#v", gotArgs[1])
	}
	if gotArgs[2] != "admin" {
		t.Errorf("arg[2]: expected positional admin, got %#v", gotArgs[2])
	}
}
