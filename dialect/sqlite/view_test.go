package sqlite

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/im"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
	_ "modernc.org/sqlite"
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

func TestCustomScannerPhysicalColumn(t *testing.T) {
	type point struct{ X, Y int }
	type model struct {
		Pos point `db:"pos"`
	}
	mapper := func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (model, error)) {
		before, after := scan.ColumnMapper[string]("pos")(ctx, cols)
		return before, func(link any) (model, error) {
			var m model
			value, err := after(link)
			if err == nil {
				_, err = fmt.Sscan(value, &m.Pos.X, &m.Pos.Y)
			}
			return m, err
		}
	}
	ctx := t.Context()
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TABLE points (pos TEXT)`); err != nil {
		t.Fatal(err)
	}
	want := model{Pos: point{1, 2}}
	for name, cols := range map[string]bob.Expression{
		"named columns": expr.NewColumnsExpr("pos").WithParent("points"),
		"expression":    Quote("points", "pos"),
	} {
		t.Run(name, func(t *testing.T) {
			tx, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := tx.Rollback(ctx); err != nil {
					t.Error(err)
				}
			})
			table := NewTablex[model, []model, setter[model]]("", "points", cols, mapper, orm.RowMapping[model, []model]{})
			inserted, err := table.Insert(im.Values(Arg("1 2"))).One(ctx, tx)
			if err != nil || inserted != want {
				t.Fatalf("INSERT RETURNING: got %+v, %v; want %+v", inserted, err, want)
			}
			selected, err := table.Query().One(ctx, tx)
			if err != nil || selected != want {
				t.Fatalf("SELECT: got %+v, %v; want %+v", selected, err, want)
			}
		})
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
