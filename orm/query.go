package orm

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

type ExecQuery[Q bob.Expression] struct {
	bob.BaseQuery[Q]
	Hooks *bob.Hooks[Q, bob.SkipQueryHooksKey]
}

func (q ExecQuery[Q]) Clone() ExecQuery[Q] {
	return ExecQuery[Q]{
		BaseQuery: q.BaseQuery.Clone(),
		Hooks:     q.Hooks,
	}
}

func (q ExecQuery[Q]) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	var err error

	ctx, err = q.BaseQuery.RunHooks(ctx, exec)
	if err != nil {
		return ctx, err
	}

	if q.Hooks == nil {
		return ctx, nil
	}

	return q.Hooks.RunHooks(ctx, exec, q.BaseQuery.Expression)
}

// Execute the query
func (q ExecQuery[Q]) Exec(ctx context.Context, exec bob.Executor) (int64, error) {
	result, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

type Query[Q bob.Expression, T any, Ts ~[]T] struct {
	ExecQuery[Q]
	Scanner scan.Mapper[T]
}

func (q Query[Q, T, Ts]) Clone() Query[Q, T, Ts] {
	return Query[Q, T, Ts]{
		ExecQuery: q.ExecQuery.Clone(),
	}
}

// First matching row
func (q Query[Q, T, Ts]) One(ctx context.Context, exec bob.Executor) (T, error) {
	return bob.One(ctx, exec, q, q.Scanner)
}

// All matching rows
func (q Query[Q, T, Ts]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	return bob.Allx[T, Ts](ctx, exec, q, q.Scanner)
}

// Cursor to scan through the results
func (q Query[Q, T, Ts]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	return bob.Cursor(ctx, exec, q, q.Scanner)
}
