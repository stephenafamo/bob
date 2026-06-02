package sqlite

import (
	"bytes"
	"context"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/expr"
)

type someStruct struct {
	ID    int64  `db:"id,pk"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

var someStructViewNoSchema = NewView[*someStruct, bob.Expression]("", "some_struct", expr.ColsForStruct[someStruct]("some_struct"))

func TestSomeViewNameAsWithoutSchema(t *testing.T) {
	q := selectToString(t, Select(sm.From(someStructViewNoSchema.NameAsExpr())), 0)

	expected := "SELECT \n*\nFROM \"some_struct\"\n"
	if q != expected {
		t.Errorf("someStructViewNoSchema.NameAs() expected '%#v' but got '%#v'", expected, q)
	}
}

func TestSomeViewQueryWithoutSchema(t *testing.T) {
	q := someStructViewNoSchema.Query(sm.Where(Quote("id").In(Arg(1, 2, 3))))
	query := viewToString(t, q)
	expected := "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"some_struct\"\nWHERE (\"id\" IN (?0, ?1, ?2))\n"

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
