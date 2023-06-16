package bob

import (
	"fmt"
	"io"
)

func replaceArgumentBindingsWithCheck(buildArgs []any, args []any) ([]any, error) {
	var nargs []ArgumentBinding
	hasNonBinding := false
	for _, buildArg := range buildArgs {
		if na, ok := buildArg.(ArgumentBinding); ok {
			nargs = append(nargs, na)
		} else {
			hasNonBinding = true
		}
	}
	if len(nargs) == 0 {
		return args, nil
	}
	if hasNonBinding {
		return nil, fmt.Errorf("cannot mix argument bindings with other arguments")
	}
	return replaceArgumentBindings(nargs, args...)
}

func MustBuildWithBinding(q Query, args ...any) (string, []any) {
	return MustBuildWithBindingN(q, 1, args...)
}

func MustBuildWithBindingN(q Query, start int, args ...any) (string, []any) {
	sql, args, err := BuildWithBindingN(q, start, args...)
	if err != nil {
		panic(err)
	}

	return sql, args
}

func BuildWithBinding(q Query, args ...any) (string, []any, error) {
	return BuildWithBindingN(q, 1, args...)
}

func BuildWithBindingN(q Query, start int, args ...any) (string, []any, error) {
	query, buildArgs, err := BuildN(q, start)
	if err != nil {
		return "", nil, err
	}

	bindArgs, err := replaceArgumentBindingsWithCheck(buildArgs, args)
	if err != nil {
		return "", nil, err
	}

	return query, bindArgs, nil
}

func QueryWithBinding(q Query, args ...any) Query {
	return &queryWithBinding{
		q:    q,
		args: args,
	}
}

type queryWithBinding struct {
	q    Query
	args []any
}

func (q queryWithBinding) WriteQuery(w io.Writer, start int) ([]any, error) {
	buildArgs, err := q.q.WriteQuery(w, start)
	if err != nil {
		return nil, err
	}
	return replaceArgumentBindingsWithCheck(buildArgs, q.args)
}

func (q queryWithBinding) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
	buildArgs, err := q.q.WriteSQL(w, d, start)
	if err != nil {
		return nil, err
	}
	return replaceArgumentBindingsWithCheck(buildArgs, q.args)
}
