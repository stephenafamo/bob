package bob

import (
	"fmt"
	"io"
)

func bindNamedArgs(buildArgs []any, args []any) ([]any, error) {
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

func BuildWithNamedArgs[E Expression](q BaseQuery[E], args ...any) (string, []any, error) {
	query, buildArgs, err := q.Build()
	if err != nil {
		return "", nil, err
	}

	bindArgs, err := bindNamedArgs(buildArgs, args)
	if err != nil {
		return "", nil, err
	}

	return query, bindArgs, nil
}

func QueryWithNamedArgs(q QueryWriter, args ...any) QueryWriter {
	return &queryWithNamedArgs{
		q:    q,
		args: args,
	}
}

type queryWithNamedArgs struct {
	q    QueryWriter
	args []any
}

func (q queryWithNamedArgs) WriteQuery(w io.Writer, start int) (args []any, err error) {
	buildArgs, err := q.q.WriteQuery(w, start)
	if err != nil {
		return nil, err
	}
	return bindNamedArgs(buildArgs, q.args)
}
