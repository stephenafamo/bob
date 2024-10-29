package mysql

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/orm"
)

type WithAutoIncr struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id,autoincr"`
}

func (w *WithAutoIncr) PrimaryKeyVals() bob.Expression {
	return Arg(w.ID)
}

type OptionalWithAutoIncr struct {
	ID       omit.Val[int]    `db:"id,pk"`
	Title    omit.Val[string] `db:"title"`
	AuthorID omit.Val[int]    `db:"author_id,autoincr"`

	orm.Setter[*WithAutoIncr, *dialect.InsertQuery, *dialect.UpdateQuery]
}

type WithUnique struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id"`
}

func (w *WithUnique) PrimaryKeyVals() bob.Expression {
	return Arg(w.ID)
}

type OptionalWithUnique struct {
	ID       omit.Val[int]    `db:"id,pk"`
	Title    omit.Val[string] `db:"title"`
	AuthorID omit.Val[int]    `db:"author_id"`

	orm.Setter[*WithUnique, *dialect.InsertQuery, *dialect.UpdateQuery]
}

var (
	table1 = NewTablex[*WithAutoIncr, []*WithAutoIncr, *OptionalWithAutoIncr]("")
	table2 = NewTablex[*WithUnique, []*WithUnique, *OptionalWithUnique](
		"", []string{"id"}, []string{"title", "author_id"},
	)
)

func TestNewTable(t *testing.T) {
	expected := "author_id"
	got := table1.autoIncrementColumn
	if got != expected {
		t.Fatalf("missing autoIncrementColumn. expected %q, got %q", expected, got)
	}

	if diff := cmp.Diff([][]int{{0}, {1, 2}}, table2.uniqueIdx); diff != "" {
		t.Fatalf("diff: %s", diff)
	}

	if table1.unretrievable {
		t.Fatalf("table1 marked as unretrievable")
	}

	if table2.unretrievable {
		t.Fatalf("table2 marked as unretrievable")
	}
}

func TestUniqueSetRow(t *testing.T) {
	cases := map[string]struct {
		row  *OptionalWithUnique
		cols []string
		args []bob.Expression
	}{
		"nil": {
			row: nil,
		},
		"none fully set": {
			row: &OptionalWithUnique{Title: omit.From("a title")},
		},
		"id set": {
			row:  &OptionalWithUnique{ID: omit.From(10)},
			cols: []string{"id"},
			args: []bob.Expression{Arg(omit.From(10))},
		},
		"title/author set": {
			row: &OptionalWithUnique{
				Title:    omit.From("a title"),
				AuthorID: omit.From(1),
			},
			cols: []string{"title", "author_id"},
			args: []bob.Expression{Arg(omit.From("a title")), Arg(omit.From(1))},
		},
		"all set": {
			row: &OptionalWithUnique{
				ID:       omit.From(10),
				Title:    omit.From("a title"),
				AuthorID: omit.From(1),
			},
			cols: []string{"id"},
			args: []bob.Expression{Arg(omit.From(10))},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rowExpr := make([]bob.Expression, 3)

			if tc.row != nil {
				if tc.row.ID.IsSet() {
					rowExpr[0] = Arg(tc.row.ID)
				}

				if tc.row.Title.IsSet() {
					rowExpr[1] = Arg(tc.row.Title)
				}

				if tc.row.AuthorID.IsSet() {
					rowExpr[2] = Arg(tc.row.AuthorID)
				}
			}

			cols, args := table2.uniqueSet(bytes.NewBuffer(nil), rowExpr)

			if diff := cmp.Diff(toQuote(tc.cols), table2.uniqueColNames(cols)); diff != "" {
				t.Errorf("cols: %s", diff)
			}

			if diff := cmp.Diff(tc.args, args, cmp.Comparer(compareArg)); diff != "" {
				t.Errorf("args: %s", diff)
			}
		})
	}
}

func toQuote(s []string) []bob.Expression {
	if len(s) == 0 {
		return nil
	}

	exprs := make([]bob.Expression, len(s))
	for i, v := range s {
		exprs[i] = Quote(v)
	}
	return exprs
}

func compareArg(a, b bob.Expression) bool {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	aArg, aErr := a.WriteSQL(ctx, buf, dialect.Dialect, 1)
	aStr := buf.String()

	buf.Reset()

	bArg, bErr := b.WriteSQL(ctx, buf, dialect.Dialect, 1)
	bStr := buf.String()

	if aErr != nil || bErr != nil {
		return false
	}

	if aStr != bStr {
		return false
	}

	if len(aArg) != len(bArg) {
		return false
	}

	for i := range aArg {
		if !reflect.DeepEqual(aArg[i], bArg[i]) {
			return false
		}
	}

	return true
}
