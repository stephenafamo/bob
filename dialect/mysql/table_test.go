package mysql

import (
	"reflect"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/google/go-cmp/cmp"
)

type WithAutoIncr struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id,autoincr"`
}

type OptionalWithAutoIncr struct {
	ID       omit.Val[int]    `db:"id,pk"`
	Title    omit.Val[string] `db:"title"`
	AuthorID omit.Val[int]    `db:"author_id,autoincr"`
}

type WithUnique struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id"`
}

type OptionalWithUnique struct {
	ID       omit.Val[int]    `db:"id,pk"`
	Title    omit.Val[string] `db:"title"`
	AuthorID omit.Val[int]    `db:"author_id"`
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

	if diff := cmp.Diff(table2.uniqueIdx, [][]int{{0}, {1, 2}}); diff != "" {
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
		args []any
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
			args: []any{omit.From(10)},
		},
		"title/author set": {
			row: &OptionalWithUnique{
				Title:    omit.From("a title"),
				AuthorID: omit.From(1),
			},
			cols: []string{"title", "author_id"},
			args: []any{omit.From("a title"), omit.From(1)},
		},
		"all set": {
			row: &OptionalWithUnique{
				ID:       omit.From(10),
				Title:    omit.From("a title"),
				AuthorID: omit.From(1),
			},
			cols: []string{"id"},
			args: []any{omit.From(10)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cols, args := table2.uniqueSet(tc.row)

			if diff := cmp.Diff(cols, tc.cols); diff != "" {
				t.Errorf("cols: %s", diff)
			}

			if diff := cmp.Diff(args, tc.args, cmp.Comparer(compareOpt)); diff != "" {
				t.Errorf("args: %s", diff)
			}
		})
	}
}

func compareOpt(a, b interface{ IsSet() bool }) bool {
	if a.IsSet() != b.IsSet() {
		return false
	}

	aVal := reflect.ValueOf(a).MethodByName("GetOrZero").Call(nil)[0].Interface()
	bVal := reflect.ValueOf(b).MethodByName("GetOrZero").Call(nil)[0].Interface()

	return aVal == bVal
}
