package bob

import (
	"context"
	"fmt"
	"io"
)

func Cache(ctx context.Context, exec Executor, q Query) (BaseQuery[*cached], error) {
	return CacheN(ctx, exec, q, 1)
}

func CacheN(ctx context.Context, exec Executor, q Query, start int) (BaseQuery[*cached], error) {
	var err error

	if h, ok := q.(HookableQuery); ok {
		ctx, err = h.RunHooks(ctx, exec)
		if err != nil {
			return BaseQuery[*cached]{}, err
		}
	}

	query, args, err := BuildN(ctx, q, start)
	if err != nil {
		return BaseQuery[*cached]{}, err
	}

	cached := BaseQuery[*cached]{
		QueryType: q.Type(),
		Expression: &cached{
			query: []byte(query),
			args:  args,
			start: start,
		},
	}

	if l, ok := q.(Loadable); ok {
		cached.Expression.SetLoaders(l.GetLoaders()...)
	}

	if m, ok := q.(MapperModder); ok {
		cached.Expression.SetMapperMods(m.GetMapperMods()...)
	}

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
	query []byte
	args  []any
	start int
	Load
}

// WriteSQL implements Expression.
func (c *cached) WriteSQL(ctx context.Context, w io.Writer, d Dialect, start int) ([]any, error) {
	if start != c.start {
		return nil, WrongStartError{Expected: c.start, Got: start}
	}

	if _, err := w.Write(c.query); err != nil {
		return nil, err
	}

	return c.args, nil
}
