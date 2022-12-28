package psql

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/select/qm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewView[T any, Tslice ~[]T](name0 string, nameX ...string) View[T, Tslice] {
	var zero T

	names := append([]string{name0}, nameX...)
	mappings := internal.GetMappings(reflect.TypeOf(zero))
	allCols := mappings.Columns(names...)

	return View[T, Tslice]{
		name:    names,
		prefix:  names[len(names)-1] + ".",
		mapping: mappings,
		allCols: allCols,
		pkCols:  allCols.Only(mappings.PKs...),
	}
}

type View[T any, Tslice ~[]T] struct {
	prefix string
	name   []string

	mapping internal.Mapping
	allCols orm.Columns
	pkCols  orm.Columns

	AfterSelectHooks orm.Hooks[T]
}

func (t *View[T, Tslice]) Name() Expression {
	return Quote(t.name...)
}

// Returns a column list
func (t *View[T, Tslice]) Columns() orm.Columns {
	return t.allCols
}

// Adds table name et al
func (t *View[T, Tslice]) Query(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := Select(qm.From(t.Name()))
	q.Apply(queryMods...)

	// Append the table columns
	if len(q.Expression.SelectList.Columns) == 0 {
		q.Expression.AppendSelect(t.Columns())
	}

	return &ViewQuery[T, Tslice]{
		BaseQuery:        q,
		afterSelectHooks: &t.AfterSelectHooks,
	}
}

// Prepare a statement that will be mapped to the view's type
func (*View[T, Tslice]) Prepare(ctx context.Context, exec bob.Preparer, q bob.Query) (bob.QueryStmt[T, Tslice], error) {
	return bob.PrepareQueryx[T, Tslice](ctx, q, scan.StructMapper[T](), exec)
}

type ViewQuery[T any, Ts ~[]T] struct {
	bob.BaseQuery[*dialect.SelectQuery]
	afterSelectHooks *orm.Hooks[T]
}

func (f *ViewQuery[T, Ts]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
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

func (f *ViewQuery[T, Tslice]) One(ctx context.Context, exec bob.Executor) (T, error) {
	f.BaseQuery.Expression.Limit.SetLimit(1)
	val, err := bob.One(ctx, exec, f.BaseQuery, scan.StructMapper[T](), f.afterSelect(ctx, exec))
	if err != nil {
		return val, err
	}

	return val, nil
}

func (f *ViewQuery[T, Tslice]) All(ctx context.Context, exec bob.Executor) (Tslice, error) {
	vals, err := bob.Allx[T, Tslice](ctx, exec, f.BaseQuery, scan.StructMapper[T](), f.afterSelect(ctx, exec))
	if err != nil {
		return nil, err
	}

	return vals, nil
}

func (f *ViewQuery[T, Tslice]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	return bob.Cursor(ctx, exec, f.BaseQuery, scan.StructMapper[T](), f.afterSelect(ctx, exec))
}

func (f *ViewQuery[T, Tslice]) Count(ctx context.Context, exec bob.Executor) (int64, error) {
	f.BaseQuery.Expression.SelectList.Columns = []any{"count(1)"}
	return bob.One(ctx, exec, f.BaseQuery, scan.SingleColumnMapper[int64])
}

func (f *ViewQuery[T, Tslice]) Exists(ctx context.Context, exec bob.Executor) (bool, error) {
	count, err := f.Count(ctx, exec)
	return count > 0, err
}

func (f *ViewQuery[T, Tslice]) Prepare(ctx context.Context, exec bob.Preparer) (bob.QueryStmt[T, Tslice], error) {
	return bob.PrepareQueryx[T, Tslice](ctx, f, scan.StructMapper[T](), exec)
}
