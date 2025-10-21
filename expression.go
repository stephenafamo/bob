package bob

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
)

var ErrNoNamedArgs = errors.New("Dialect does not support named arguments")

// Dialect provides expressions with methods to write parts of the query
type Dialect interface {
	// WriteArg should write an argument placeholder to the writer with the given index
	WriteArg(w io.StringWriter, position int)

	// WriteQuoted writes the given string to the writer surrounded by the appropriate
	// quotes for the dialect
	WriteQuoted(w io.StringWriter, s string)
}

// DialectWithNamed is a [Dialect] with the additional ability to WriteNamedArgs
type DialectWithNamed interface {
	Dialect
	// WriteNamedArg should write an argument placeholder to the writer with the given name
	WriteNamedArg(w io.StringWriter, name string)
}

// Expression represents a section of a query
type Expression interface {
	// Writes the textual representation of the expression to the writer
	// using the given dialect.
	// start is the beginning index of the args if it needs to write any
	WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) (args []any, err error)
}

type ExpressionFunc func(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error)

func (e ExpressionFunc) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	return e(ctx, w, d, start)
}

func Express(ctx context.Context, w io.StringWriter, d Dialect, start int, e any) ([]any, error) {
	switch v := e.(type) {
	case string:
		w.WriteString(v)
		return nil, nil
	case []byte:
		w.WriteString(string(v))
		return nil, nil
	case sql.NamedArg:
		dn, ok := d.(DialectWithNamed)
		if !ok {
			return nil, ErrNoNamedArgs
		}
		dn.WriteNamedArg(w, v.Name)
		return []any{v}, nil
	case Expression:
		return v.WriteSQL(ctx, w, d, start)
	default:
		w.WriteString(fmt.Sprint(v))
		return nil, nil
	}
}

// ExpressIf expands an express if the condition evaluates to true
// it can also add a prefix and suffix
func ExpressIf(ctx context.Context, w io.StringWriter, d Dialect, start int, e any, cond bool, prefix, suffix string) ([]any, error) {
	if !cond {
		return nil, nil
	}

	w.WriteString(prefix)
	args, err := Express(ctx, w, d, start, e)
	if err != nil {
		return nil, err
	}
	w.WriteString(suffix)

	return args, nil
}

// ExpressSlice is used to express a slice of expressions along with a prefix and suffix
func ExpressSlice[T any](ctx context.Context, w io.StringWriter, d Dialect, start int, expressions []T, prefix, sep, suffix string) ([]any, error) {
	if len(expressions) == 0 {
		return nil, nil
	}

	var args []any
	w.WriteString(prefix)

	for k, e := range expressions {
		if k != 0 {
			w.WriteString(sep)
		}

		newArgs, err := Express(ctx, w, d, start+len(args), e)
		if err != nil {
			return args, err
		}

		args = append(args, newArgs...)
	}
	w.WriteString(suffix)

	return args, nil
}
