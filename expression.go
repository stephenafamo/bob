package bob

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
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

type sqlWriterTo interface {
	WriteSQLTo(ctx context.Context, w io.StringWriter, d Dialect, start int, args *[]any) error
}

func Express(ctx context.Context, w io.StringWriter, d Dialect, start int, e any) ([]any, error) {
	var args []any
	err := ExpressTo(ctx, w, d, start, e, &args)
	return args, err
}

func ExpressTo(ctx context.Context, w io.StringWriter, d Dialect, start int, e any, args *[]any) error {
	switch v := e.(type) {
	case string:
		w.WriteString(v)
		return nil
	case []byte:
		w.WriteString(string(v))
		return nil
	case bool:
		w.WriteString(strconv.FormatBool(v))
		return nil
	case int:
		w.WriteString(strconv.Itoa(v))
		return nil
	case int8:
		w.WriteString(strconv.FormatInt(int64(v), 10))
		return nil
	case int16:
		w.WriteString(strconv.FormatInt(int64(v), 10))
		return nil
	case int32:
		w.WriteString(strconv.FormatInt(int64(v), 10))
		return nil
	case int64:
		w.WriteString(strconv.FormatInt(v, 10))
		return nil
	case uint:
		w.WriteString(strconv.FormatUint(uint64(v), 10))
		return nil
	case uint8:
		w.WriteString(strconv.FormatUint(uint64(v), 10))
		return nil
	case uint16:
		w.WriteString(strconv.FormatUint(uint64(v), 10))
		return nil
	case uint32:
		w.WriteString(strconv.FormatUint(uint64(v), 10))
		return nil
	case uint64:
		w.WriteString(strconv.FormatUint(v, 10))
		return nil
	case float32:
		w.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
		return nil
	case float64:
		w.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
		return nil
	case sql.NamedArg:
		dn, ok := d.(DialectWithNamed)
		if !ok {
			return ErrNoNamedArgs
		}
		dn.WriteNamedArg(w, v.Name)
		*args = append(*args, v)
		return nil
	case sqlWriterTo:
		return v.WriteSQLTo(ctx, w, d, start, args)
	case Expression:
		newArgs, err := v.WriteSQL(ctx, w, d, start)
		*args = MergeArgs(*args, newArgs)
		return err
	case interface{ String() string }:
		w.WriteString(v.String())
		return nil
	default:
		w.WriteString(fmt.Sprint(v))
		return nil
	}
}

// ExpressIf expands an express if the condition evaluates to true
// it can also add a prefix and suffix
func ExpressIf(ctx context.Context, w io.StringWriter, d Dialect, start int, e any, cond bool, prefix, suffix string) ([]any, error) {
	var args []any
	err := ExpressIfTo(ctx, w, d, start, e, cond, prefix, suffix, &args)
	return args, err
}

func ExpressIfTo(ctx context.Context, w io.StringWriter, d Dialect, start int, e any, cond bool, prefix, suffix string, args *[]any) error {
	if !cond {
		return nil
	}

	w.WriteString(prefix)
	if err := ExpressTo(ctx, w, d, start, e, args); err != nil {
		return err
	}
	w.WriteString(suffix)

	return nil
}

// ExpressSlice is used to express a slice of expressions along with a prefix and suffix
func ExpressSlice[T any](ctx context.Context, w io.StringWriter, d Dialect, start int, expressions []T, prefix, sep, suffix string) ([]any, error) {
	var args []any
	err := ExpressSliceTo(ctx, w, d, start, expressions, prefix, sep, suffix, &args)
	return args, err
}

func ExpressSliceTo[T any](ctx context.Context, w io.StringWriter, d Dialect, start int, expressions []T, prefix, sep, suffix string, args *[]any) error {
	if len(expressions) == 0 {
		return nil
	}

	baseLen := len(*args)
	w.WriteString(prefix)

	for k, e := range expressions {
		if k != 0 {
			w.WriteString(sep)
		}

		if err := ExpressTo(ctx, w, d, start+len(*args)-baseLen, e, args); err != nil {
			return err
		}
	}
	w.WriteString(suffix)

	return nil
}

func MergeArgs(dst, src []any) []any {
	switch {
	case len(src) == 0:
		return dst
	case len(dst) == 0:
		return src
	default:
		return append(dst, src...)
	}
}
