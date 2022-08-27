package model

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type ViewQuery[T any, Ts ~[]T] struct {
	bob.BaseQuery[*psql.SelectQuery]
	afterSelectHooks *orm.Hooks[T]
}

func (f *ViewQuery[T, Ts]) afterSelect(ctx context.Context, exec scan.Queryer) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			for _, val := range retrieved {
				_, err := f.afterSelectHooks.Do(ctx, exec, val)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}
}

func (f *ViewQuery[T, Tslice]) One(ctx context.Context, exec scan.Queryer) (T, error) {
	f.BaseQuery.Expression.Limit.SetLimit(1)
	val, err := bob.One(ctx, exec, f.BaseQuery, scan.StructMapper[T](), f.afterSelect(ctx, exec))
	if err != nil {
		return val, err
	}

	return val, nil
}

func (f *ViewQuery[T, Tslice]) All(ctx context.Context, exec scan.Queryer) (Tslice, error) {
	vals, err := bob.Allx[T, Tslice](ctx, exec, f.BaseQuery, scan.StructMapper[T](), f.afterSelect(ctx, exec))
	if err != nil {
		return nil, err
	}

	return vals, nil
}

func (f *ViewQuery[T, Tslice]) Cursor(ctx context.Context, exec scan.Queryer) (scan.ICursor[T], error) {
	return bob.Cursor(ctx, exec, f.BaseQuery, scan.StructMapper[T](), f.afterSelect(ctx, exec))
}

func (f *ViewQuery[T, Tslice]) Count(ctx context.Context, exec scan.Queryer) (int64, error) {
	f.BaseQuery.Expression.Select.Columns = []any{"count(1)"}
	return bob.One(ctx, exec, f.BaseQuery, scan.SingleColumnMapper[int64])
}

func (f *ViewQuery[T, Tslice]) Exists(ctx context.Context, exec scan.Queryer) (bool, error) {
	f.BaseQuery.Expression.Select.Columns = []any{"count(1)"}
	count, err := bob.One(ctx, exec, f.BaseQuery, scan.SingleColumnMapper[int64])
	return count > 0, err
}

type TableQuery[T any, Ts ~[]T, Topt any] struct {
	ViewQuery[T, Ts]
}

func (f *TableQuery[T, Tslice, Topt]) UpdateAll(Topt) (int64, error) {
	panic("not implemented")
}

func (f *TableQuery[T, Tslice, Topt]) DeleteAll() (int64, error) {
	panic("not implemented")
}
