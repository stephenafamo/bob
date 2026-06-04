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
// "SELECT id FROM todo", mirroring what the queries plugin emits.
func newModQuery(scanner scan.Mapper[int]) orm.ModQuery[*dialect.SelectQuery, rawExpr, int, []int, bob.SliceTransformer[int, []int]] {
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
