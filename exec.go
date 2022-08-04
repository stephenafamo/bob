package bob

import (
	"context"

	"github.com/stephenafamo/bob/scanto"
)

func One[T any](ctx context.Context, exec scanto.Queryer, q Query, m scanto.MapperGen[T]) (T, error) {
	var t T

	sql, args, err := Build(q)
	if err != nil {
		return t, err
	}

	return scanto.One[T](ctx, exec, m, sql, args...)
}

func All[T any](ctx context.Context, exec scanto.Queryer, q Query, m scanto.MapperGen[T]) ([]T, error) {
	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	return scanto.All[T](ctx, exec, m, sql, args...)
}
