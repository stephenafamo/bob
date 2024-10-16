package bob

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"strconv"
	"testing"
)

var d dialect

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	w.Write([]byte("$"))
	w.Write([]byte(strconv.Itoa(position)))
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	w.Write([]byte(`"`))
	w.Write([]byte(s))
	w.Write([]byte(`"`))
}

var expression = ExpressionFunc(func(ctx context.Context, w io.Writer, d Dialect, start int) ([]any, error) {
	w.Write([]byte("Hello "))
	d.WriteArg(w, start)
	w.Write([]byte(" "))
	d.WriteArg(w, start+1)
	return nil, nil
})

func compare(t *testing.T, sqlExpected, sqlGotten string, argsExpected, argsGotten []any) {
	t.Helper()

	if sqlExpected != sqlGotten {
		t.Fatalf("Wrong sql string\nExpected: %s\nGot: %s", sqlExpected, sqlGotten)
	}

	if len(argsGotten) != len(argsExpected) {
		t.Fatalf("wrong length of debug args.\nExpected: %d\nGot: %d\n\n%s", len(argsExpected), len(argsGotten), argsGotten)
	}

	for i := range argsExpected {
		arg := argsExpected[i]
		debugArg := argsGotten[i]
		if arg != debugArg {
			t.Fatalf("wrong debug arg %d.\nExpected: %#v\nGot: %#v", i, arg, debugArg)
		}
	}
}

func TestExpress(t *testing.T) {
	w := bytes.NewBuffer(nil)
	args, err := Express(context.Background(), w, d, 2, expression)
	if err != nil {
		t.Fatalf("err while expressing")
	}

	compare(t, "Hello $2 $3", w.String(), nil, args)
}

func TestExpress2(t *testing.T) {
	tests := map[string]struct {
		value    any
		expected string
	}{
		"string": {
			value:    "a string",
			expected: "a string",
		},
		"[]byte": {
			value:    []byte("a byte slice"),
			expected: "a byte slice",
		},
		"Expression": {
			value:    expression,
			expected: "Hello $1 $2",
		},
		"number": {
			value:    100,
			expected: "100",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			args, err := Express(context.Background(), w, d, 1, test.value)
			if err != nil {
				t.Fatalf("err while expressing")
			}

			compare(t, test.expected, w.String(), nil, args)
		})
	}
}

func TestExpressIf(t *testing.T) {
	tests := map[string]struct {
		cond   bool
		prefix string
		suffix string
	}{
		"false": {
			cond: false,
		},
		"false with prefix and suffix": {
			cond:   false,
			prefix: "pr",
			suffix: "sf",
		},
		"true": {
			cond: true,
		},
		"true with prefix and suffix": {
			cond:   true,
			prefix: "pr",
			suffix: "sf",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			args, err := ExpressIf(context.Background(), w, d, 1, expression, test.cond, test.prefix, test.suffix)
			if err != nil {
				t.Fatalf("err while expressing")
			}

			expected := test.prefix + "Hello $1 $2" + test.suffix
			if !test.cond {
				expected = ""
			}

			compare(t, expected, w.String(), nil, args)
		})
	}
}

func TestExpressEmptySlice(t *testing.T) {
	w := bytes.NewBuffer(nil)
	args, err := ExpressSlice(context.Background(), w, d, 2, []string{}, "prefix ", ", ", " suffix")
	if err != nil {
		t.Fatalf("err while expressing")
	}

	compare(t, "", w.String(), nil, args)
}

func TestExpressSlice(t *testing.T) {
	w := bytes.NewBuffer(nil)
	args, err := ExpressSlice(context.Background(), w, d, 2, []string{"one", "two", "three"}, "prefix ", ", ", " suffix")
	if err != nil {
		t.Fatalf("err while expressing")
	}

	compare(t, "prefix one, two, three suffix", w.String(), nil, args)
}

type dialectWithNamed struct{ dialect }

func (d dialectWithNamed) WriteNamedArg(w io.Writer, name string) {
	w.Write([]byte(":"))
	w.Write([]byte(name))
}

func TestNamedArgs(t *testing.T) {
	arg := sql.Named("name", "value")
	w := bytes.NewBuffer(nil)
	args, err := Express(context.Background(), w, dialectWithNamed{}, 1, arg)
	if err != nil {
		t.Fatalf("err while expressing")
	}

	compare(t, ":name", w.String(), []any{arg}, args)
}

func TestErrNoNamedArgs(t *testing.T) {
	arg := sql.Named("name", "value")
	w := bytes.NewBuffer(nil)
	_, err := Express(context.Background(), w, d, 1, arg)
	if !errors.Is(err, ErrNoNamedArgs) {
		t.Fatalf("Expected to get ErrNoNamedArgs but got %v", err)
	}
}
