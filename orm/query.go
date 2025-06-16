package orm

import (
	"context"
	"fmt"
	"io"
	"iter"

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

type Query[Q bob.Expression, T, Ts any, Tr bob.Transformer[T, Ts]] struct {
	ExecQuery[Q]
	Scanner scan.Mapper[T]
}

func (q Query[Q, T, Ts, Tr]) Clone() Query[Q, T, Ts, Tr] {
	return Query[Q, T, Ts, Tr]{
		ExecQuery: q.ExecQuery.Clone(),
	}
}

// First matching row
func (q Query[Q, T, Ts, Tr]) One(ctx context.Context, exec bob.Executor) (T, error) {
	return bob.One(ctx, exec, q, q.Scanner)
}

// All matching rows
func (q Query[Q, T, Ts, Tr]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	return bob.Allx[Tr](ctx, exec, q, q.Scanner)
}

// Cursor to scan through the results
func (q Query[Q, T, Ts, Tr]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	return bob.Cursor(ctx, exec, q, q.Scanner)
}

type ModExecQuery[Q any, E bob.Expression] struct {
	ExecQuery[E]
	Mod bob.Mod[Q]
}

func (q ModExecQuery[Q, E]) Apply(e Q) {
	q.Mod.Apply(e)
}

type ModQuery[Q any, E bob.Expression, T, Ts any, Tr bob.Transformer[T, Ts]] struct {
	Query[E, T, Ts, Tr]
	Mod bob.Mod[Q]
}

func (q ModQuery[Q, E, T, Ts, Tr]) Apply(e Q) {
	q.Mod.Apply(e)
}

func ArgsToExpression(querySQL string, from, to int, argIter iter.Seq[ArgWithPosition]) bob.Expression {
	return bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
		args := []any{}

		for queryArg := range argIter {
			if to < queryArg.Start {
				w.Write([]byte(querySQL[from:to]))
				return args, nil
			}

			if from > queryArg.Start {
				if to < queryArg.Stop {
					return nil, fmt.Errorf("arg %q end(%d) is after expression end(%d)", queryArg.Name, queryArg.Stop, to)
				}
				continue
			}

			if to < queryArg.Stop {
				return nil, fmt.Errorf("arg %q end(%d) is greater than to(%d)", queryArg.Name, queryArg.Stop, to)
			}

			w.Write([]byte(querySQL[from:queryArg.Start]))

			arg, err := bob.Express(ctx, w, d, start, queryArg.Expression)
			if err != nil {
				return nil, err
			}
			args = append(args, arg...)

			start += len(arg)
			from = queryArg.Stop
		}

		w.Write([]byte(querySQL[from:to]))
		return args, nil
	})
}
