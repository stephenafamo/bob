package model

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/scan"
)

type ViewQuery[T any, Ts ~[]T] struct {
	bob.BaseQuery[*psql.SelectQuery]
}

func (f *ViewQuery[T, Tslice]) One(ctx context.Context, exec scan.Queryer) (T, error) {
	f.BaseQuery.Expression.Limit.SetLimit(1)
	return bob.One(ctx, exec, f.BaseQuery, scan.StructMapper[T]())
}

func (f *ViewQuery[T, Tslice]) All(ctx context.Context, exec scan.Queryer) (Tslice, error) {
	return bob.Allx[T, Tslice](ctx, exec, f.BaseQuery, scan.StructMapper[T]())
}

func (f *ViewQuery[T, Tslice]) Cursor(ctx context.Context, exec scan.Queryer) (scan.ICursor[T], error) {
	return bob.Cursor(ctx, exec, f.BaseQuery, scan.StructMapper[T]())
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
