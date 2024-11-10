package orm

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

type expression interface {
	bob.Expression
	RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error)
}

type ExecQuery[Q expression, T any, Ts ~[]T] struct {
	bob.BaseQuery[Q]
	Scanner scan.Mapper[T]
	Hooks   *bob.Hooks[Q, bob.SkipQueryHooksKey]
}

func (q ExecQuery[Q, T, Ts]) Clone() ExecQuery[Q, T, Ts] {
	return ExecQuery[Q, T, Ts]{
		BaseQuery: q.BaseQuery.Clone(),
		Scanner:   q.Scanner,
		Hooks:     q.Hooks,
	}
}

func (q ExecQuery[Q, T, Ts]) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	ctx, err := q.Expression.RunHooks(ctx, exec)
	if err != nil {
		return ctx, err
	}

	if q.Hooks == nil {
		return ctx, nil
	}

	return q.Hooks.RunHooks(ctx, exec, q.BaseQuery.Expression)
}

// Execute the query
func (q ExecQuery[Q, T, Ts]) Exec(ctx context.Context, exec bob.Executor) (int64, error) {
	result, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

type Query[Q expression, T any, Ts ~[]T] struct {
	ExecQuery[Q, T, Ts]
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
