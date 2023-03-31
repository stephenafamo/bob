package bob

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
)

var ErrNoNamedArgs = errors.New("Dialect does not support named arguments")

// Dialect provides expressions with methods to write parts of the query
type Dialect interface {
	// WriteArg should write an argument placeholder to the writer with the given index
	WriteArg(w io.Writer, position int)

	// WriteQuoted writes the given string to the writer surrounded by the appropriate
	// quotes for the dialect
	WriteQuoted(w io.Writer, s string)
}

// DialectWithNamed is a [Dialect] with the additional ability to WriteNamedArgs
type DialectWithNamed interface {
	Dialect
	// WriteNamedArg should write an argument placeholder to the writer with the given name
	WriteNamedArg(w io.Writer, name string)
}

// Expression represents a section of a query
type Expression interface {
	// Writes the textual representation of the expression to the writer
	// using the given dialect.
	// start is the beginning index of the args if it needs to write any
	WriteSQL(w io.Writer, d Dialect, start int) (args []any, err error)
}

type ExpressionFunc func(w io.Writer, d Dialect, start int) ([]any, error)

func (e ExpressionFunc) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
	return e(w, d, start)
}

func Express(w io.Writer, d Dialect, start int, e any) ([]any, error) {
	switch v := e.(type) {
	case string:
		w.Write([]byte(v))
		return nil, nil
	case []byte:
		w.Write(v)
		return nil, nil
	case sql.NamedArg:
		dn, ok := d.(DialectWithNamed)
		if !ok {
			return nil, ErrNoNamedArgs
		}
		dn.WriteNamedArg(w, v.Name)
		return []any{v}, nil
	case Expression:
		return v.WriteSQL(w, d, start)
	default:
		fmt.Fprint(w, e)
		return nil, nil
	}
}

// ExpressIf expands an express if the condition evaluates to true
// it can also add a prefix and suffix
func ExpressIf(w io.Writer, d Dialect, start int, e any, cond bool, prefix, suffix string) ([]any, error) {
	if !cond {
		return nil, nil
	}

	w.Write([]byte(prefix))
	args, err := Express(w, d, start, e)
	if err != nil {
		return nil, err
	}
	w.Write([]byte(suffix))

	return args, nil
}

// ExpressSlice is used to express a slice of expressions along with a prefix and suffix
// a converter can also be passed to modify each expression before expressing it.
func ExpressSlice[T any](w io.Writer, d Dialect, start int, expressions []T, prefix, sep, suffix string, converter ...func(T) any) ([]any, error) {
	if len(expressions) == 0 {
		return nil, nil
	}

	c := defaultConverter[T]
	if len(converter) > 0 {
		c = converter[0]
	}

	var args []any
	w.Write([]byte(prefix))

	for k, e := range expressions {
		if k != 0 {
			w.Write([]byte(sep))
		}

		newArgs, err := Express(w, d, start+len(args), c(e))
		if err != nil {
			return args, err
		}

		args = append(args, newArgs...)
	}
	w.Write([]byte(suffix))

	return args, nil
}

func defaultConverter[T any](e T) any {
	return e
}
