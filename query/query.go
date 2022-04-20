package query

import (
	"bytes"
	"fmt"
	"io"
)

// MustBuild builds a query and panics on error
// useful for initializing queries that need to be reused
func MustBuild(q Query) (string, []any) {
	return MustBuildN(q, 1)
}

func MustBuildN(q Query, start int) (string, []any) {
	sql, args, err := BuildN(q, start)
	if err != nil {
		panic(err)
	}

	return sql, args
}

// Convinient function to build query from start
func Build(q Query) (string, []any, error) {
	return BuildN(q, 1)
}

// Convinient function to build query from a point
func BuildN(q Query, start int) (string, []any, error) {
	b := &bytes.Buffer{}
	args, err := q.WriteQuery(b, start)

	return b.String(), args, err
}

type Query interface {
	// start is the index of the args, usually 1.
	// it is present to allow re-indexing in cases of a subquery
	// The method returns the value of any args placed
	WriteQuery(w io.Writer, start int) (args []any, err error)
}

type Expression interface {
	// Like query, but takes a dialect
	WriteSQL(w io.Writer, d Dialect, start int) (args []any, err error)
}

type Dialect interface {
	WriteArg(w io.Writer, position int)
	WriteQuoted(w io.Writer, s string)
}

func Express(w io.Writer, d Dialect, start int, e any) ([]any, error) {
	switch v := e.(type) {
	case string:
		w.Write([]byte(v))
		return nil, nil
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
func ExpressSlice[T any](w io.Writer, d Dialect, start int, expressions []T, prefix, sep, suffix string) ([]any, error) {
	if len(expressions) == 0 {
		return nil, nil
	}

	var args []any
	w.Write([]byte(prefix))

	for k, e := range expressions {
		if k != 0 {
			w.Write([]byte(sep))
		}

		newArgs, err := Express(w, d, start+len(args), e)
		if err != nil {
			return args, err
		}

		args = append(args, newArgs...)
	}
	w.Write([]byte(suffix))

	return args, nil
}
