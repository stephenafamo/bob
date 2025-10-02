package psql

import (
	"bytes"
	"context"
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

var someStructView = NewView[*someStruct, bob.Expression]("public", "some_struct", expr.ColsForStruct[someStruct](""), expr.ColsForStruct[someStruct]("some_struct"))

func TestSomeViewName(t *testing.T) {
	name := someStructView.Name().String()
	expected := "\"public\".\"some_struct\""
	if name != expected {
		t.Errorf("someStructView.Name() expected '%s' but got '%s'", expected, name)
	}
}

func TestSomeViewNameAs(t *testing.T) {
	q := selectToString(t, Select(sm.From(someStructView.NameAs())), 0)

	expected := "SELECT \n*\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\n"
	if q != expected {
		t.Errorf("someStructView.NameAs() expected '%#v' but got '%#v'", expected, q)
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

func TestSomeViewColumnNames(t *testing.T) {
	c := someStructView.ColumnNames
	query := selectToString(t, Select(sm.Columns(c)), 0)
	expected := "SELECT \n\"id\" AS \"id\", \"name\" AS \"name\", \"email\" AS \"email\"\n"
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
