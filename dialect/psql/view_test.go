package psql

import (
	"bytes"
	"context"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
)

type someStruct struct {
	ID    int64  `db:"id,pk"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

var someStructView = NewView[*someStruct, bob.Expression]("public", "some_struct", expr.ColsForStruct[someStruct]("some_struct"))

var someStructViewNoSchema = NewView[*someStruct, bob.Expression]("", "some_struct", expr.ColsForStruct[someStruct]("some_struct"))

func TestSomeViewName(t *testing.T) {
	name := someStructView.NameExpr().String()
	expected := "\"public\".\"some_struct\""
	if name != expected {
		t.Errorf("someStructView.Name() expected '%s' but got '%s'", expected, name)
	}
}

func TestSomeViewNameAs(t *testing.T) {
	q := selectToString(t, Select(sm.From(someStructView.NameAsExpr())), 0)

	expected := "SELECT \n*\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\n"
	if q != expected {
		t.Errorf("someStructView.NameAs() expected '%#v' but got '%#v'", expected, q)
	}
}

func TestSomeViewNameAsWithoutSchema(t *testing.T) {
	q := selectToString(t, Select(sm.From(someStructViewNoSchema.NameAsExpr())), 0)

	expected := "SELECT \n*\nFROM \"some_struct\"\n"
	if q != expected {
		t.Errorf("someStructViewNoSchema.NameAs() expected '%#v' but got '%#v'", expected, q)
	}
}

func TestSomeViewColumns(t *testing.T) {
	c := someStructView.Columns
	query := selectToString(t, Select(sm.Columns(c)), 0)
	expected := "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\n"
	if query != expected {
		t.Errorf("Expected '%#v' but got '%#v'", expected, query)
	}
}

func TestSomeViewQuery(t *testing.T) {
	q := someStructView.Query(sm.Where(Quote("id").In(Arg(1, 2, 3))))
	query := viewToString(t, q)
	expected := "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" IN ($0, $1, $2))\n"

	if query != expected {
		t.Errorf("Expected '%#v' but got '%#v'", expected, query)
	}
}

func TestSomeViewQueryWithoutSchema(t *testing.T) {
	q := someStructViewNoSchema.Query(sm.Where(Quote("id").In(Arg(1, 2, 3))))
	query := viewToString(t, q)
	expected := "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"some_struct\"\nWHERE (\"id\" IN ($0, $1, $2))\n"

	if query != expected {
		t.Errorf("Expected '%#v' but got '%#v'", expected, query)
	}
}

func TestSomeViewExistsUsesExistsExpression(t *testing.T) {
	ctx := context.Background()
	if _, err := testDB.ExecContext(ctx, `CREATE TABLE exists_view_test (id BIGINT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := testDB.ExecContext(ctx, `DROP TABLE exists_view_test`); err != nil {
			t.Error(err)
		}
	})
	if _, err := testDB.ExecContext(ctx, `INSERT INTO exists_view_test (id) VALUES (1)`); err != nil {
		t.Fatal(err)
	}

	view := NewView[int64, bob.Expression]("", "exists_view_test", Quote("id"))
	var query strings.Builder
	exists, err := view.Query(
		sm.With("matching").As(Select(sm.Columns("id"), sm.From("exists_view_test"))),
		sm.From("matching"),
		sm.Where(Quote("id").EQ(Arg(1))),
	).Exists(ctx, bob.DebugToWriter(testDB, &query))
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected matching row to exist")
	}
	got := query.String()
	existsAt := strings.Index(got, "EXISTS ((")
	withAt := strings.Index(got, "WITH")
	switch {
	case existsAt < 0:
		t.Fatalf("expected an EXISTS subquery, got:\n%s", got)
	case withAt < 0:
		t.Fatalf("expected the CTE WITH clause to be preserved, got:\n%s", got)
	case withAt < existsAt:
		// A WITH hoisted ahead of EXISTS would change semantics for
		// data-modifying CTEs; keeping it inside the subquery is the property
		// the EXISTS optimization relies on for CTE-backed views.
		t.Fatalf("expected the WITH clause to stay inside the EXISTS subquery, got:\n%s", got)
	case strings.Contains(got, "count(1)"):
		t.Fatalf("expected an EXISTS probe rather than count(1), got:\n%s", got)
	}
}

func selectToString(t *testing.T, query bob.BaseQuery[*dialect.SelectQuery], argsLen int) string {
	t.Helper()
	ctx := context.Background()
	buf := new(bytes.Buffer)
	args, err := query.WriteQuery(ctx, buf, 0)
	if err != nil {
		t.Errorf("Failed to WriteQuery: %v", err)
		return ""
	}
	if len(args) != argsLen {
		t.Errorf("Expected %d args but got %d", argsLen, len(args))
		return ""
	}
	return buf.String()
}

func viewToString(t *testing.T, query *ViewQuery[*someStruct, []*someStruct]) string {
	t.Helper()
	return selectToString(t, query.BaseQuery, 3)
}
