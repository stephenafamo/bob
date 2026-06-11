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

// newModQuery builds a ModQuery whose generated Mod reproduces
// "SELECT id FROM todo", mirroring what the queries plugin emits. Extra funcs
// run inside the Mod after the SELECT/FROM, so a test can add the clauses a
// generated query already carries.
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

// TestModQueryWithExistingClauses augments a generated query that already
// carries clauses: WHERE conditions are ANDed, ORDER BY merges into one clause,
// and LIMIT / OFFSET are replaced.
func TestModQueryWithExistingClauses(t *testing.T) {
	examples := testutils.Testcases{
		"existing WHERE is ANDed with one extra condition": {
			Doc:          "A user sm.Where() ANDs onto the existing WHERE",
			ExpectedSQL:  `SELECT id FROM todo WHERE done = $1 AND project_id = $2`,
			ExpectedArgs: []any{true, 1},
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendWhere(psql.Quote("done").EQ(psql.Arg(true)))
			}).With(
				sm.Where(psql.Quote("project_id").EQ(psql.Arg(1))),
			),
		},
		"existing WHERE is ANDed with multiple extra conditions": {
			Doc:          "Multiple user sm.Where() mods each AND on, preserving arg order",
			ExpectedSQL:  `SELECT id FROM todo WHERE done = $1 AND project_id = $2 AND priority > $3`,
			ExpectedArgs: []any{true, 1, 2},
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendWhere(psql.Quote("done").EQ(psql.Arg(true)))
			}).With(
				sm.Where(psql.Quote("project_id").EQ(psql.Arg(1))),
				sm.Where(psql.Quote("priority").GT(psql.Arg(2))),
			),
		},
		"existing ORDER BY merges with the user ORDER BY": {
			Doc:         "A user sm.OrderBy() appends into the existing ORDER BY clause",
			ExpectedSQL: `SELECT id FROM todo ORDER BY created_at, id`,
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendOrder(psql.Quote("created_at"))
			}).With(
				sm.OrderBy(psql.Quote("id")),
			),
		},
		"existing LIMIT is replaced by the user LIMIT": {
			Doc:         "A user sm.Limit() replaces the existing LIMIT",
			ExpectedSQL: `SELECT id FROM todo LIMIT 10`,
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.SetLimit(5)
			}).With(
				sm.Limit(10),
			),
		},
		"existing OFFSET is replaced by the user OFFSET": {
			Doc:         "A user sm.Offset() replaces the existing OFFSET",
			ExpectedSQL: `SELECT id FROM todo OFFSET 10`,
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.SetOffset(5)
			}).With(
				sm.Offset(10),
			),
		},
		"WHERE ANDs, ORDER BY merges, LIMIT is replaced": {
			Doc:          "Combined scenario across all three behaviors",
			ExpectedSQL:  `SELECT id FROM todo WHERE done = $1 AND project_id = $2 ORDER BY created_at, id LIMIT 10`,
			ExpectedArgs: []any{true, 1},
			Query: newModQuery(nil, func(q *dialect.SelectQuery) {
				q.AppendWhere(psql.Quote("done").EQ(psql.Arg(true)))
				q.AppendOrder(psql.Quote("created_at"))
				q.SetLimit(5)
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
