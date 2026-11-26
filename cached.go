package bob

import (
	"context"
	"fmt"
	"io"
	"slices"
)

// CachedQuery holds SQL and arguments built for a particular context and start index.
type CachedQuery = ExpressionQuery[*cached]

func Cache(ctx context.Context, exec Executor, q Query) (CachedQuery, error) {
	return CacheN(ctx, exec, q, 1)
}

func CacheN(ctx context.Context, exec Executor, q Query, start int) (CachedQuery, error) {
	ctx, q, err := resolveForExecution(ctx, exec, q)
	if err != nil {
		return CachedQuery{}, err
	}

	query, args, err := render(ctx, exec, q, start)
	if err != nil {
		return ExpressionQuery[*cached]{}, err
	}

	cached := QueryFromExpression(&cached{
		query: query,
		args:  slices.Clone(args),
		start: start,
	}, nil, q.Type())

	cached.expression.SetLoaders(loadersFor(ctx, q)...)
	cached.expression.SetMapperMods(mapperModsFor(ctx, q)...)

	return cached, nil
}

type WrongStartError struct {
	Expected int
	Got      int
}

func (e WrongStartError) Error() string {
	return fmt.Sprintf("expected to start at %d, started at %d", e.Expected, e.Got)
}

type cached struct {
	query string
	args  []any
	start int
	Load
}

// WriteSQL implements Expression.
func (c *cached) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	if start != c.start {
		return nil, WrongStartError{Expected: c.start, Got: start}
	}

	if _, err := w.WriteString(c.query); err != nil {
		return nil, err
	}

	return slices.Clone(c.args), nil
}
