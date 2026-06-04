package orm_test

import (
	"context"
	"io"
	"testing"

	"github.com/stephenafamo/scan"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/orm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

type rawExpr struct{ sql string }

func (r rawExpr) WriteSQL(_ context.Context, w io.StringWriter, _ bob.Dialect, _ int) ([]any, error) {
	w.WriteString(r.sql)
	return nil, nil
}

// newModQuery builds a ModQuery whose generated Mod alone reproduces
// "SELECT id FROM todo", mirroring what the queries plugin emits. Any extra
// funcs are applied inside the generated Mod after the SELECT/FROM, letting a
// test reproduce a query that already carries WHERE/ORDER BY/LIMIT/OFFSET
// clauses the way the generator emits them.
func newModQuery(scanner scan.Mapper[int], extra ...func(*dialect.SelectQuery)) orm.ModQuery[*dialect.SelectQuery, rawExpr, int, []int, bob.SliceTransformer[int, []int]] {
	return orm.ModQuery[*dialect.SelectQuery, rawExpr, int, []int, bob.SliceTransformer[int, []int]]{
		Query: orm.Query[rawExpr, int, []int, bob.SliceTransformer[int, []int]]{
			ExecQuery: orm.ExecQuery[rawExpr]{
				BaseQuery: bob.BaseQuery[rawExpr]{
					Expression: rawExpr{sql: "SELECT id FROM todo"},
					Dialect:    dialect.Dialect,
					QueryType:  bob.QueryTypeSelect,
				},
			},
			Scanner: scanner,
		},
		Mod: bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
			q.AppendSelect(psql.Quote("id"))
			q.SetTable(psql.Quote("todo"))
			for _, fn := range extra {
				fn(q)
			}
		}),
		Build: psql.Select,
	}
}

func TestModQueryWith(t *testing.T) {
	mq := newModQuery(nil)

	examples := testutils.Testcases{
		"base query from generated mod": {
			Doc:         "With() and no extra mods reproduces the base query from the generated Mod alone",
			ExpectedSQL: `SELECT id FROM todo`,
			Query:       mq.With(),
		},
		"augmented with extra mods": {
			Doc:          "Extra mods are appended on top of the generated Mod",
			ExpectedSQL:  `SELECT id FROM todo WHERE (project_id = $1) LIMIT 10`,
			ExpectedArgs: []any{1},
			Query: mq.With(
				sm.Where(psql.Quote("project_id").EQ(psql.Arg(1))),
				sm.Limit(10),
			),
		},
	}

	testutils.RunTests(t, examples, nil)
}

// TestModQueryWithExistingClauses exercises augmentation of a generated query
// that already carries WHERE / ORDER BY / LIMIT / OFFSET clauses, reproducing
// the way the queries plugin emits them: WHERE goes onto the regular Where
// field (so user mods AND onto it), while ORDER BY / LIMIT / OFFSET go onto the
// Combined* fields.
func TestModQueryWithExistingClauses(t *testing.T) {
	examples := testutils.Testcases{
		"existing WHERE is ANDed with one extra condition": {
			Doc:          "A user sm.Where() is appended to the generated WHERE via AND, producing valid SQL",
			ExpectedSQL:  `SELECT id FROM todo WHERE done = $1 AND project_id = $2`,
			ExpectedArgs: []any{true, 1},
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendWhere(psql.Quote("done").EQ(psql.Arg(true)))
			}).With(
				sm.Where(psql.Quote("project_id").EQ(psql.Arg(1))),
			),
		},
		"existing WHERE is ANDed with multiple extra conditions": {
			Doc:          "Multiple user sm.Where() mods each AND onto the generated WHERE, preserving arg order",
			ExpectedSQL:  `SELECT id FROM todo WHERE done = $1 AND project_id = $2 AND priority > $3`,
			ExpectedArgs: []any{true, 1, 2},
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendWhere(psql.Quote("done").EQ(psql.Arg(true)))
			}).With(
				sm.Where(psql.Quote("project_id").EQ(psql.Arg(1))),
				sm.Where(psql.Quote("priority").GT(psql.Arg(2))),
			),
		},
		// KNOWN LIMITATION: ORDER BY / LIMIT / OFFSET only append. The generated
		// clauses live on the Combined* fields while user mods set the regular
		// fields, so both are rendered as separate clauses, yielding invalid SQL.
		"existing ORDER BY produces a duplicate clause (invalid SQL)": {
			Doc:         "A user sm.OrderBy() does not merge with the generated ORDER BY; both clauses are emitted",
			ExpectedSQL: `SELECT id FROM todo ORDER BY id ORDER BY created_at`,
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.CombinedOrder.AppendOrder(psql.Quote("created_at"))
			}).With(
				sm.OrderBy(psql.Quote("id")),
			),
		},
		"existing LIMIT produces a duplicate clause (invalid SQL)": {
			Doc:         "A user sm.Limit() does not replace the generated LIMIT; both clauses are emitted",
			ExpectedSQL: `SELECT id FROM todo LIMIT 10 LIMIT 5`,
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.CombinedLimit.SetLimit(5)
			}).With(
				sm.Limit(10),
			),
		},
		"existing OFFSET produces a duplicate clause (invalid SQL)": {
			Doc:         "A user sm.Offset() does not replace the generated OFFSET; both clauses are emitted",
			ExpectedSQL: `SELECT id FROM todo OFFSET 10 OFFSET 5`,
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.CombinedOffset.SetOffset(5)
			}).With(
				sm.Offset(10),
			),
		},
		"WHERE merges while ORDER BY and LIMIT duplicate": {
			Doc:          "Combined scenario: the WHERE is correctly ANDed, but ORDER BY and LIMIT each duplicate",
			ExpectedSQL:  `SELECT id FROM todo WHERE done = $1 AND project_id = $2 ORDER BY id LIMIT 10 ORDER BY created_at LIMIT 5`,
			ExpectedArgs: []any{true, 1},
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendWhere(psql.Quote("done").EQ(psql.Arg(true)))
				q.CombinedOrder.AppendOrder(psql.Quote("created_at"))
				q.CombinedLimit.SetLimit(5)
			}).With(
				sm.Where(psql.Quote("project_id").EQ(psql.Arg(1))),
				sm.OrderBy(psql.Quote("id")),
				sm.Limit(10),
			),
		},
	}

	testutils.RunTests(t, examples, nil)
}

func TestModQueryWithPreservesScanner(t *testing.T) {
	scannerCalled := false
	scanner := func(context.Context, []string) (func(*scan.Row) (any, error), func(any) (int, error)) {
		scannerCalled = true
		return nil, nil
	}

	augmented := newModQuery(scanner).With()
	if augmented.Scanner == nil {
		t.Fatal("With() dropped the scanner")
	}

	augmented.Scanner(context.Background(), nil)
	if !scannerCalled {
		t.Fatal("With() did not preserve the original scanner")
	}
}
