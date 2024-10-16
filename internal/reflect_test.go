package internal

import (
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/mappings"
)

type User struct {
	ID        int
	FirstName string
	LastName  string
}

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserWithTimestamps struct {
	ID        string
	FirstName string
	LastName  string
	Timestamps
}

type Blog struct {
	ID          string
	Title       string
	Description string
	User        User
}

type BlogWithTags struct {
	ID          string `db:"blog_id,pk,generated,autoincr"`
	Title       string `db:"title,pk"`
	Description string `db:"description,generated"`
	User        User   `db:"-"`
}

func TestGetColumns(t *testing.T) {
	testGetColumns[User](t, mappings.Mapping{
		All:           []string{"id", "first_name", "last_name"},
		PKs:           make([]string, 3),
		NonPKs:        []string{"id", "first_name", "last_name"},
		Generated:     make([]string, 3),
		NonGenerated:  []string{"id", "first_name", "last_name"},
		AutoIncrement: make([]string, 3),
	})

	testGetColumns[Timestamps](t, mappings.Mapping{
		All:           []string{"created_at", "updated_at"},
		PKs:           make([]string, 2),
		NonPKs:        []string{"created_at", "updated_at"},
		Generated:     make([]string, 2),
		NonGenerated:  []string{"created_at", "updated_at"},
		AutoIncrement: make([]string, 2),
	})

	testGetColumns[UserWithTimestamps](t, mappings.Mapping{
		All:           []string{"id", "first_name", "last_name", "timestamps"},
		PKs:           make([]string, 4),
		NonPKs:        []string{"id", "first_name", "last_name", "timestamps"},
		Generated:     make([]string, 4),
		NonGenerated:  []string{"id", "first_name", "last_name", "timestamps"},
		AutoIncrement: make([]string, 4),
	})

	testGetColumns[Blog](t, mappings.Mapping{
		All:           []string{"id", "title", "description", "user"},
		PKs:           make([]string, 4),
		NonPKs:        []string{"id", "title", "description", "user"},
		Generated:     make([]string, 4),
		NonGenerated:  []string{"id", "title", "description", "user"},
		AutoIncrement: make([]string, 4),
	})

	testGetColumns[BlogWithTags](t, mappings.Mapping{
		All:           []string{"blog_id", "title", "description", ""},
		PKs:           []string{"blog_id", "title", "", ""},
		NonPKs:        []string{"", "", "description", ""},
		Generated:     []string{"blog_id", "", "description", ""},
		NonGenerated:  []string{"", "title", "", ""},
		AutoIncrement: []string{"blog_id", "", "", ""},
	})
}

func testGetColumns[T any](t *testing.T, expected mappings.Mapping) {
	t.Helper()
	var x T
	xTyp := reflect.TypeOf(x)
	t.Run(xTyp.Name(), func(t *testing.T) {
		cols := mappings.GetMappings(xTyp)
		if diff := cmp.Diff(expected, cols); diff != "" {
			t.Fatal(diff)
		}
	})
}

type SettableUser struct {
	ID        int
	FirstName string
	LastName  string
	FullName  string `db:",generated"`
	Bio       omit.Val[string]
}

type testGetColumnsCase[T any] struct {
	Filter  []string
	Rows    []T
	Columns []string
	Values  [][]bob.Expression
}

func TestGetColumnValues(t *testing.T) {
	user1 := User{ID: 1, FirstName: "Stephen", LastName: "AfamO"}
	user2 := User{ID: 2, FirstName: "Peter", LastName: "Pan"}
	user3 := SettableUser{ID: 3, FirstName: "John", LastName: "Doe", FullName: "John Doe"}
	user4 := SettableUser{
		ID: 4, FirstName: "Jane", LastName: "Does",
		FullName: "Jane Does", Bio: omit.From("Foo Bar"),
	}

	testGetColumnValues(t, "pointer", testGetColumnsCase[*User]{
		Rows:    []*User{&user1},
		Columns: []string{"id", "first_name", "last_name"},
		Values:  [][]bob.Expression{{expr.Arg(1), expr.Arg("Stephen"), expr.Arg("AfamO")}},
	})

	testGetColumnValues(t, "single row", testGetColumnsCase[User]{
		Rows:    []User{user1},
		Columns: []string{"id", "first_name", "last_name"},
		Values:  [][]bob.Expression{{expr.Arg(1), expr.Arg("Stephen"), expr.Arg("AfamO")}},
	})

	testGetColumnValues(t, "with generated", testGetColumnsCase[SettableUser]{
		Rows:    []SettableUser{user3},
		Columns: []string{"id", "first_name", "last_name"},
		Values:  [][]bob.Expression{{expr.Arg(3), expr.Arg("John"), expr.Arg("Doe")}},
	})

	testGetColumnValues(t, "settable and set", testGetColumnsCase[SettableUser]{
		Rows:    []SettableUser{user4},
		Columns: []string{"id", "first_name", "last_name", "bio"},
		Values: [][]bob.Expression{{
			expr.Arg(4), expr.Arg("Jane"),
			expr.Arg("Does"), expr.Arg(omit.From("Foo Bar")),
		}},
	})

	testGetColumnValues(t, "single user with filter", testGetColumnsCase[User]{
		Filter:  []string{"first_name", "last_name"},
		Rows:    []User{user1},
		Columns: []string{"first_name", "last_name"},
		Values:  [][]bob.Expression{{expr.Arg("Stephen"), expr.Arg("AfamO")}},
	})

	testGetColumnValues(t, "multiple users", testGetColumnsCase[User]{
		Rows:    []User{user1, user2},
		Columns: []string{"id", "first_name", "last_name"},
		Values: [][]bob.Expression{
			{expr.Arg(1), expr.Arg("Stephen"), expr.Arg("AfamO")},
			{expr.Arg(2), expr.Arg("Peter"), expr.Arg("Pan")},
		},
	})

	testGetColumnValues(t, "multiple users with filter", testGetColumnsCase[User]{
		Filter:  []string{"id", "first_name"},
		Rows:    []User{user1, user2},
		Columns: []string{"id", "first_name"},
		Values: [][]bob.Expression{
			{expr.Arg(1), expr.Arg("Stephen")},
			{expr.Arg(2), expr.Arg("Peter")},
		},
	})

	testGetColumnValues(t, "first has fewer columns", testGetColumnsCase[SettableUser]{
		Rows:    []SettableUser{user3, user4},
		Columns: []string{"id", "first_name", "last_name"},
		Values: [][]bob.Expression{
			{expr.Arg(3), expr.Arg("John"), expr.Arg("Doe")},
			{expr.Arg(4), expr.Arg("Jane"), expr.Arg("Does")},
		},
	})

	testGetColumnValues(t, "first has more columns", testGetColumnsCase[SettableUser]{
		Rows:    []SettableUser{user4, user3},
		Columns: []string{"id", "first_name", "last_name", "bio"},
		Values: [][]bob.Expression{
			{expr.Arg(4), expr.Arg("Jane"), expr.Arg("Does"), expr.Arg(omit.From("Foo Bar"))},
			{expr.Arg(3), expr.Arg("John"), expr.Arg("Doe"), expr.Arg(omit.FromCond("", false))},
		},
	})
}

func testGetColumnValues[T any](t *testing.T, name string, tc testGetColumnsCase[T]) {
	t.Helper()
	var x T
	xTyp := reflect.TypeOf(x)
	cols := mappings.GetMappings(xTyp)
	if name == "" {
		name = xTyp.Name()
	}

	t.Run(name, func(t *testing.T) {
		cols, values, err := GetColumnValues(cols, tc.Filter, tc.Rows...)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(tc.Columns, cols); diff != "" {
			t.Errorf("Columns: %s", diff)
		}
		if diff := cmp.Diff(tc.Values, values,
			cmp.Transformer("optTransformer", optTransformer),
			cmp.Transformer("expTransformer", expTransformer),
		); diff != "" {
			t.Errorf("Values: %s", diff)
		}
	})
}

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	fmt.Fprintf(w, "$%s", strconv.Itoa(position))
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	fmt.Fprintf(w, "%q", s)
}

type expression struct {
	Query string
	Args  []any
	Error error
}

func expTransformer(e bob.Expression) expression {
	buf := &bytes.Buffer{}
	args, err := e.WriteSQL(context.Background(), buf, dialect{}, 1)

	return expression{
		Query: buf.String(),
		Args:  args,
		Error: err,
	}
}

func optTransformer(e interface{ IsSet() bool }) any {
	if v, ok := e.(driver.Valuer); ok {
		value, err := v.Value()
		return []any{value, err}
	}

	return []any{e.IsSet(), fmt.Sprint(e)}
}
