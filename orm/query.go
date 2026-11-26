package orm

import (
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/stephenafamo/scan"

	"github.com/stephenafamo/bob"
)

type ExecQuery[Q any] struct {
	bob.QueryBuilder[Q]
}

func (q ExecQuery[Q]) WithMods(mods ...bob.Mod[Q]) *ExecQuery[Q] {
	q.QueryBuilder = q.QueryBuilder.WithMods(mods...)
	return &q
}

// Execute the query
func (q ExecQuery[Q]) Exec(ctx context.Context, exec bob.Executor) (int64, error) {
	result, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

type Query[Q any, T, Ts any, Tr bob.Transformer[T, Ts]] struct {
	ExecQuery[Q]
	Scanner scan.Mapper[T]
}

func (q Query[Q, T, Ts, Tr]) WithMods(mods ...bob.Mod[Q]) *Query[Q, T, Ts, Tr] {
	q.QueryBuilder = q.QueryBuilder.WithMods(mods...)
	return &q
}

// WithScanner uses a different row mapper for this branch.
func (q Query[Q, T, Ts, Tr]) WithScanner(mapper scan.Mapper[T]) *Query[Q, T, Ts, Tr] {
	q.Scanner = mapper
	return &q
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

// Each to scan through the results
func (q Query[Q, T, Ts, Tr]) Each(ctx context.Context, exec bob.Executor) (func(func(T, error) bool), error) {
	return bob.Each(ctx, exec, q, q.Scanner)
}

type ModExecQuery[Q any, E bob.Expression] struct {
	ExecQuery[Q]
	Mod        bob.Mod[Q]
	NewBuilder func(...bob.Mod[Q]) bob.QueryBuilder[Q]
}

func (q ModExecQuery[Q, E]) Edit(e Q) {
	q.Mod.Edit(e)
}

func (q ModExecQuery[Q, E]) With(mods ...bob.Mod[Q]) *ExecQuery[Q] {
	return &ExecQuery[Q]{
		QueryBuilder: q.NewBuilder(append([]bob.Mod[Q]{q.Mod}, mods...)...),
	}
}

type ModQuery[Q any, E bob.Expression, T, Ts any, Tr bob.Transformer[T, Ts]] struct {
	Query[Q, T, Ts, Tr]
	Mod        bob.Mod[Q]
	NewBuilder func(...bob.Mod[Q]) bob.QueryBuilder[Q]
}

func (q ModQuery[Q, E, T, Ts, Tr]) Edit(e Q) {
	q.Mod.Edit(e)
}

func (q ModQuery[Q, E, T, Ts, Tr]) With(mods ...bob.Mod[Q]) *Query[Q, T, Ts, Tr] {
	return &Query[Q, T, Ts, Tr]{
		ExecQuery: ExecQuery[Q]{
			QueryBuilder: q.NewBuilder(append([]bob.Mod[Q]{q.Mod}, mods...)...),
		},
		Scanner: q.Scanner,
	}
}

func ArgsToExpression(querySQL string, from, to int, argIter iter.Seq[ArgWithPosition]) bob.Expression {
	return bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		cursor := from
		args := []any{}

		for queryArg := range argIter {
			if to <= queryArg.Start {
				w.WriteString(querySQL[cursor:to])
				return args, nil
			}

			if queryArg.Stop <= cursor {
				continue
			}

			if queryArg.Start < cursor {
				return nil, fmt.Errorf("arg %q starts before expression start(%d)", queryArg.Name, cursor)
			}
			if to < queryArg.Stop {
				return nil, fmt.Errorf("arg %q end(%d) is greater than to(%d)", queryArg.Name, queryArg.Stop, to)
			}

			w.WriteString(querySQL[cursor:queryArg.Start])

			arg, err := bob.Express(ctx, w, d, start, queryArg.Expression)
			if err != nil {
				return nil, err
			}
			args = append(args, arg...)

			start += len(arg)
			cursor = queryArg.Stop
		}

		w.WriteString(querySQL[cursor:to])
		return args, nil
	})
}
