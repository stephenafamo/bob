package bob

import (
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type customString string

type binderTester interface {
	Run(t *testing.T, origin []any)
}

type binderTests[Arg any] struct {
	args  Arg
	final []any
	err   error
}

func (s binderTests[Arg]) Run(t *testing.T, origin []any) {
	t.Helper()

	t.Run("", func(t *testing.T) {
		binder, err := makeBinder[Arg](origin)
		if !errors.Is(err, s.err) {
			t.Fatal(err)
		}

		if s.err != nil {
			return
		}

		if diff := cmp.Diff(
			s.final, binder.toArgs(s.args), cmpopts.EquateEmpty(),
		); diff != "" {
			t.Fatal(diff)
		}
	})
}

func testBinder(t *testing.T, origin []any, tests []binderTester) {
	t.Helper()

	for _, test := range tests {
		test.Run(t, origin)
	}
}

func TestBinding(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		testBinder(t, []any{}, []binderTester{
			binderTests[struct{}]{
				args:  struct{}{},
				final: []any{},
			},
			binderTests[map[customString]any]{
				args:  nil,
				final: []any{},
			},
			binderTests[int]{},
		})
	})

	t.Run("no named", func(t *testing.T) {
		testBinder(t, []any{1, 2, 3, 4}, []binderTester{
			binderTests[struct{}]{
				args:  struct{}{},
				final: []any{1, 2, 3, 4},
			},
			binderTests[map[string]any]{
				args:  nil,
				final: []any{1, 2, 3, 4},
			},
			binderTests[int]{
				args:  0,
				final: []any{1, 2, 3, 4},
			},
		})
	})

	t.Run("all named", func(t *testing.T) {
		testBinder(t, []any{namedArg("one"), namedArg("two"), namedArg("three"), namedArg("four")}, []binderTester{
			binderTests[struct{ One, Two, Three, Four int }]{
				args: struct{ One, Two, Three, Four int }{
					One: 1, Two: 2, Three: 3, Four: 4,
				},
				final: []any{1, 2, 3, 4},
			},
			binderTests[map[string]int]{
				args:  map[string]int{"one": 1, "two": 2, "three": 3, "four": 4},
				final: []any{1, 2, 3, 4},
			},
			binderTests[int]{
				err: ErrBadArgType,
			},
		})
	})

	t.Run("mixed named", func(t *testing.T) {
		testBinder(t, []any{1, 2, namedArg("three"), 4}, []binderTester{
			binderTests[struct{ Three int }]{
				args:  struct{ Three int }{Three: 3},
				final: []any{1, 2, 3, 4},
			},
			binderTests[map[string]int]{
				args:  map[string]int{"three": 3},
				final: []any{1, 2, 3, 4},
			},
			binderTests[int]{
				args:  3,
				final: []any{1, 2, 3, 4},
			},
		})
	})

	t.Run("mixed named with nil arg", func(t *testing.T) {
		testBinder(t, []any{1, 2, namedArg("three"), 4}, []binderTester{
			binderTests[*struct{ Three int }]{
				args:  nil,
				final: []any{1, 2, nil, 4},
			},
			binderTests[map[string]int]{
				args:  nil,
				final: []any{1, 2, nil, 4},
			},
			binderTests[*int]{
				args:  nil,
				final: []any{1, 2, (*int)(nil), 4},
			},
		})
	})

	t.Run("varaitions of single binder", func(t *testing.T) {
		timeVal, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
		if err != nil {
			t.Fatal(err)
		}

		testBinder(t, []any{1, 2, namedArg("three"), 4}, []binderTester{
			binderTests[int]{
				args:  3,
				final: []any{1, 2, 3, 4},
			},
			binderTests[*int]{
				args:  nil,
				final: []any{1, 2, (*int)(nil), 4},
			},
			binderTests[time.Time]{
				args:  timeVal,
				final: []any{1, 2, timeVal, 4},
			},
			binderTests[valuable]{
				args:  valuable{3},
				final: []any{1, 2, valuable{3}, 4},
			},
		})
	})

	t.Run("reuse names", func(t *testing.T) {
		testBinder(t, []any{1, 2, namedArg("three"), 4, namedArg("three")}, []binderTester{
			binderTests[struct{ Three int }]{
				args:  struct{ Three int }{Three: 3},
				final: []any{1, 2, 3, 4, 3},
			},
			binderTests[map[string]int]{
				args:  map[string]int{"three": 3},
				final: []any{1, 2, 3, 4, 3},
			},
			binderTests[int]{
				args:  3,
				final: []any{1, 2, 3, 4, 3},
			},
		})
	})
}

type valuable struct {
	val int
}

func (v valuable) Value() (driver.Value, error) {
	return v.val, nil
}

func (v valuable) Equal(other valuable) bool {
	return v.val == other.val
}
