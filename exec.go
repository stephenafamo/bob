package bob

import (
	"context"

	"github.com/stephenafamo/scan"
)

func One[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) (T, error) {
	var t T

	sql, args, err := Build(q)
	if err != nil {
		return t, err
	}

	return scan.One(ctx, exec, m, sql, args...)
}

func All[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) ([]T, error) {
	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	return scan.All(ctx, exec, m, sql, args...)
}

// Cursor returns a cursor that works similar to *sql.Rows
func Cursor[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) (scan.ICursor[T], error) {
	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	return scan.Cursor(ctx, exec, m, sql, args...)
}

// Collect multiple slices of values from a single query
// collector must be of the structure
// func(cols) func(*Values) (t1, t2, ..., error)
// The returned slice contains values like this
// {[]t1, []t2}
func Collect(ctx context.Context, exec scan.Queryer, q Query, collector func(cols map[string]int) any) ([]any, error) {
	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	return scan.Collect(ctx, exec, collector, sql, args...)
}
