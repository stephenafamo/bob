package bob

import (
	"bytes"
	"context"
	"io"
)

// MustBuild builds a query and panics on error
// useful for initializing queries that need to be reused
func MustBuild(ctx context.Context, exec Executor, q Query) (string, []any) {
	return MustBuildN(ctx, exec, q, 1)
}

func MustBuildN(ctx context.Context, exec Executor, q Query, start int) (string, []any) {
	sql, args, err := BuildN(ctx, exec, q, start)
	if err != nil {
		panic(err)
	}

	return sql, args
}

// Build runs query hooks and renders SQL. Pass nil if no executor is
// available. The supplied executor is also used by hooks on nested queries.
func Build(ctx context.Context, exec Executor, q Query) (string, []any, error) {
	return BuildN(ctx, exec, q, 1)
}

// BuildN is Build with an explicit starting placeholder index.
func BuildN(ctx context.Context, exec Executor, q Query, start int) (string, []any, error) {
	ctx, q, err := resolveForBuild(ctx, exec, q)
	if err != nil {
		return "", nil, err
	}
	return render(ctx, exec, q, start)
}

type queryWriter struct {
	bytes.Buffer
	exec Executor
}

// render writes an already resolved outer query. Its children resolve their
// own hooks as they reach the writer, with the same executor and argument index.
func render(ctx context.Context, exec Executor, q Query, start int) (string, []any, error) {
	b := &queryWriter{exec: exec}
	args, err := q.WriteQuery(ctx, b, start)

	return b.String(), args, err
}

func writerExecutor(w io.StringWriter) Executor {
	if b, ok := w.(*queryWriter); ok {
		return b.exec
	}
	return nil
}

// WriteQuery writes a nested statement without adding parentheses. Use the
// enclosing writer to retain the build executor for its query hooks.
func WriteQuery(ctx context.Context, w io.StringWriter, start int, q Query) ([]any, error) {
	ctx, q, err := resolveForBuild(ctx, writerExecutor(w), q)
	if err != nil {
		return nil, err
	}
	return q.WriteQuery(ctx, w, start)
}
