package bob

import (
	"fmt"
	"io"
)

func BindNamedArgs(buildArgs []any, args []any) ([]any, error) {
	var nargs []NamedArgument
	hasNonNamed := false
	for _, buildArg := range buildArgs {
		if na, ok := buildArg.(NamedArgument); ok {
			nargs = append(nargs, na)
		} else {
			hasNonNamed = true
		}
	}
	if len(nargs) == 0 {
		return args, nil
	}
	if hasNonNamed {
		return nil, fmt.Errorf("cannot mix named and non-named arguments")
	}
	return mergeNamedArguments(nargs, args...)
}

func MustBuildWithNamedArgs(q Query, args ...any) (string, []any) {
	return MustBuildNWithNamedArgs(q, 1, args...)
}

func MustBuildNWithNamedArgs(q Query, start int, args ...any) (string, []any) {
	sql, args, err := BuildNWithNamedArgs(q, start, args...)
	if err != nil {
		panic(err)
	}

	return sql, args
}

func BuildWithNamedArgs(q Query, args ...any) (string, []any, error) {
	return BuildNWithNamedArgs(q, 1, args...)
}

func BuildNWithNamedArgs(q Query, start int, args ...any) (string, []any, error) {
	query, buildArgs, err := BuildN(q, start)
	if err != nil {
		return "", nil, err
	}

	bindArgs, err := BindNamedArgs(buildArgs, args)
	if err != nil {
		return "", nil, err
	}

	return query, bindArgs, nil
}

func QueryWithNamedArgs(q Query, args ...any) Query {
	return &queryWithNamedArgs{
		q:    q,
		args: args,
	}
}

type queryWithNamedArgs struct {
	q    Query
	args []any
}

func (q queryWithNamedArgs) WriteQuery(w io.Writer, start int) ([]any, error) {
	buildArgs, err := q.q.WriteQuery(w, start)
	if err != nil {
		return nil, err
	}
	return BindNamedArgs(buildArgs, q.args)
}

func (q queryWithNamedArgs) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
	buildArgs, err := q.q.WriteSQL(w, d, start)
	if err != nil {
		return nil, err
	}
	return BindNamedArgs(buildArgs, q.args)
}
