package bob

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func (s structBinder[Arg]) Equal() {
}

type binderTests[Arg any] struct {
	arg   Arg
	final []any
}

type structBinderTest[Arg any] struct {
	expected structBinder[Arg]
	args     []binderTests[Arg]
}

func testBinder[Arg any](t *testing.T, origin []any, tests []binderTests[Arg]) {
	t.Helper()
	binder, err := makeStructBinder[Arg](origin)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		if diff := cmp.Diff(binder.ToArgs(test.arg), test.final); diff != "" {
			t.Fatal(diff)
		}
	}
}

func TestStructBinding(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		testBinder(t, []any{}, []binderTests[struct{}]{
			{
				arg:   struct{}{},
				final: []any{},
			},
		})
	})

	t.Run("no named", func(t *testing.T) {
		testBinder(t, []any{1, 2, 3, 4}, []binderTests[struct{}]{
			{
				arg:   struct{}{},
				final: []any{1, 2, 3, 4},
			},
		})
	})

	t.Run("all named", func(t *testing.T) {
		testBinder(t, []any{namedArg("one"), namedArg("two"), namedArg("three"), namedArg("four")}, []binderTests[struct{ One, Two, Three, Four int }]{
			{
				arg:   struct{ One, Two, Three, Four int }{One: 1, Two: 2, Three: 3, Four: 4},
				final: []any{1, 2, 3, 4},
			},
		})
	})

	t.Run("mixed named", func(t *testing.T) {
		testBinder(t, []any{1, 2, namedArg("three"), 4}, []binderTests[struct {
			Three int
		}]{
			{
				arg:   struct{ Three int }{Three: 3},
				final: []any{1, 2, 3, 4},
			},
		})
	})
}
