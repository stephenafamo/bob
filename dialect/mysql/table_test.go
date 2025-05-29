package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
)

type WithAutoIncr struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id,autoincr"`
}

type OptionalWithAutoIncr struct {
	ID       *int    `db:"id,pk"`
	Title    *string `db:"title"`
	AuthorID *int    `db:"author_id,autoincr"`

	orm.Setter[*WithAutoIncr, *dialect.InsertQuery, *dialect.UpdateQuery]
}

type WithUnique struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id"`
}

type OptionalWithUnique struct {
	ID       *int    `db:"id,pk"`
	Title    *string `db:"title"`
	AuthorID *int    `db:"author_id"`

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

	if err := table1.Insert().retrievable(); err != nil {
		t.Fatalf("table1 marked as unretrievable: %v", err)
	}

	if err := table2.Insert().retrievable(); err != nil {
		t.Fatalf("table2 marked as unretrievable: %v", err)
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
			row: &OptionalWithUnique{Title: internal.Pointer("a title")},
		},
		"id set": {
			row:  &OptionalWithUnique{ID: internal.Pointer(10)},
			cols: []string{"id"},
			args: []bob.Expression{Arg(internal.Pointer(10))},
		},
		"title/author set": {
			row: &OptionalWithUnique{
				Title:    internal.Pointer("a title"),
				AuthorID: internal.Pointer(1),
			},
			cols: []string{"title", "author_id"},
			args: []bob.Expression{Arg(internal.Pointer("a title")), Arg(internal.Pointer(1))},
		},
		"all set": {
			row: &OptionalWithUnique{
				ID:       internal.Pointer(10),
				Title:    internal.Pointer("a title"),
				AuthorID: internal.Pointer(1),
			},
			cols: []string{"id"},
			args: []bob.Expression{Arg(internal.Pointer(10))},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rowExpr := make([]bob.Expression, 3)

			if tc.row != nil {
				if tc.row.ID != nil {
					rowExpr[0] = Arg(tc.row.ID)
				}

				if tc.row.Title != nil {
					rowExpr[1] = Arg(tc.row.Title)
				}

				if tc.row.AuthorID != nil {
					rowExpr[2] = Arg(tc.row.AuthorID)
				}
			}

			cols, args := table2.Insert().uniqueSet(bytes.NewBuffer(nil), rowExpr)

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

func TestIsDefaultOrNull(t *testing.T) {
	cases := map[string]struct {
		value  bob.Expression // value to check
		expect bool
	}{
		"nil": {
			value:  nil,
			expect: true,
		},
		"nil Arg": {
			value:  Arg(nil),
			expect: true,
		},
		"sql.NullString": {
			value:  Arg(sql.NullString{}),
			expect: true,
		},
		"sql.NullStringTrue": {
			value:  Arg(sql.NullString{Valid: true}),
			expect: false,
		},
		"sql.Null[string]": {
			value:  Arg(sql.Null[string]{}),
			expect: true,
		},
		"sql.Null[string]True": {
			value:  Arg(sql.Null[string]{Valid: true}),
			expect: false,
		},
		"null expression": {
			value:  Raw("null"),
			expect: true,
		},
		"null expression capital": {
			value:  Raw("NULL"),
			expect: true,
		},
		"default expression": {
			value:  Raw("DEFAULT"),
			expect: true,
		},
		"int zero": {
			value:  Arg(0),
			expect: false,
		},
		"int non-zero": {
			value:  Arg(1),
			expect: false,
		},
		"string empty": {
			value:  Arg(""),
			expect: false,
		},
		"string non-empty": {
			value:  Arg("hello"),
			expect: false,
		},
	}

	b := &bytes.Buffer{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isDefaultOrNull(b, tc.value); got != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}
