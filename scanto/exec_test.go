package scanto

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func createDB(tb testing.TB, cols [][2]string) db {
	tb.Helper()
	db, err := Open("test", "foo")
	if err != nil {
		tb.Fatalf("Error opening testdb %v", err)
	}

	first := true
	b := &strings.Builder{}
	fmt.Fprintf(b, "CREATE|%s|", tb.Name())

	for _, def := range cols {
		if !first {
			b.WriteString(",")
		} else {
			first = false
		}

		fmt.Fprintf(b, "%s=%s", def[0], def[1])
	}

	exec(tb, db, b.String())
	return db
}

func exec(tb testing.TB, exec db, query string, args ...interface{}) sql.Result {
	tb.Helper()
	result, err := exec.ExecContext(context.Background(), query, args...)
	if err != nil {
		tb.Fatalf("Exec of %q: %v", query, err)
	}

	return result
}

func insert(tb testing.TB, ex db, cols []string, vals ...[]any) {
	tb.Helper()
	query := fmt.Sprintf("INSERT|%s|%s=?", tb.Name(), strings.Join(cols, "=?,"))
	for _, val := range vals {
		exec(tb, ex, query, val...)
	}
}

func createQuery(tb testing.TB, cols []string) string {
	tb.Helper()
	return fmt.Sprintf("SELECT|%s|%s|", tb.Name(), strings.Join(cols, ","))
}

type (
	strstr = [][2]string
	rows   = [][]any
)

type queryCase[T any] struct {
	columns     strstr
	rows        rows
	query       []string // columns to select
	mapper      MapperGen[T]
	expectOne   T
	expectAll   []T
	expectedErr error
}

func testQuery[T any](t *testing.T, name string, tc queryCase[T]) {
	t.Helper()
	ctx := context.Background()

	t.Run(name, func(t *testing.T) {
		ex := createDB(t, tc.columns)
		insert(t, ex, colSliceFromMap(tc.columns), tc.rows...)
		query := createQuery(t, tc.query)

		t.Run("one", func(t *testing.T) {
			one, err := One(ctx, ex, tc.mapper, query)
			if diff := cmp.Diff(tc.expectedErr, err); diff != "" {
				t.Fatalf("diff: %s", diff)
			}

			if diff := cmp.Diff(tc.expectOne, one); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})

		t.Run("all", func(t *testing.T) {
			all, err := All(ctx, ex, tc.mapper, query)
			if diff := cmp.Diff(tc.expectedErr, err); diff != "" {
				t.Fatalf("diff: %s", diff)
			}

			if diff := cmp.Diff(tc.expectAll, all); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	})
}

func TestSingleValue(t *testing.T) {
	testQuery(t, "int", queryCase[int]{
		columns:   strstr{{"id", "int64"}},
		rows:      singleRows(1, 2, 3, 5, 8, 13, 21),
		query:     []string{"id"},
		mapper:    SingleValueMapper[int],
		expectOne: 1,
		expectAll: []int{1, 2, 3, 5, 8, 13, 21},
	})

	testQuery(t, "string", queryCase[string]{
		columns:   strstr{{"name", "string"}},
		rows:      singleRows("first", "second", "third"),
		query:     []string{"name"},
		mapper:    SingleValueMapper[string],
		expectOne: "first",
		expectAll: []string{"first", "second", "third"},
	})

	time1 := randate()
	time2 := randate()
	time3 := randate()
	testQuery(t, "datetime", queryCase[time.Time]{
		columns:   strstr{{"when", "datetime"}},
		rows:      singleRows(time1, time2, time3),
		query:     []string{"when"},
		mapper:    SingleValueMapper[time.Time],
		expectOne: time1,
		expectAll: []time.Time{time1, time2, time3},
	})
}

func TestStruct(t *testing.T) {
	user1 := User{ID: 1, Name: "foo"}
	user2 := User{ID: 2, Name: "bar"}

	testQuery(t, "user", queryCase[User]{
		columns:   strstr{{"id", "int64"}, {"name", "string"}},
		rows:      rows{[]any{1, "foo"}, []any{2, "bar"}},
		query:     []string{"id", "name"},
		mapper:    StructMapper[User],
		expectOne: user1,
		expectAll: []User{user1, user2},
	})

	createdAt1 := randate()
	createdAt2 := randate()
	updatedAt1 := randate()
	updatedAt2 := randate()
	timestamp1 := &Timestamps{CreatedAt: createdAt1, UpdatedAt: updatedAt1}
	timestamp2 := &Timestamps{CreatedAt: createdAt2, UpdatedAt: updatedAt2}

	testQuery(t, "userwithtimestamps", queryCase[UserWithTimestamps]{
		columns: strstr{
			{"id", "int64"},
			{"name", "string"},
			{"created_at", "datetime"},
			{"updated_at", "datetime"},
		},
		rows: rows{
			[]any{1, "foo", createdAt1, updatedAt1},
			[]any{2, "bar", createdAt2, updatedAt2},
		},
		query:     []string{"id", "name", "created_at", "updated_at"},
		mapper:    StructMapper[UserWithTimestamps],
		expectOne: UserWithTimestamps{User: user1, Timestamps: timestamp1},
		expectAll: []UserWithTimestamps{
			{User: user1, Timestamps: timestamp1},
			{User: user2, Timestamps: timestamp2},
		},
	})
}
