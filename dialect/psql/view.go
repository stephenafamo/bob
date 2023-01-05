package psql

import (
	"context"
	"errors"
	"io"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/select/qm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

var ErrNotPreparer = errors.New("executor provided is not a bob.Preparer")

func NewView[T any, Tslice ~[]T](schema, tableName string) *View[T, Tslice] {
	v, _ := newView[T, Tslice](schema, tableName)
	return v
}

func newView[T any, Tslice ~[]T](schema, tableName string) (*View[T, Tslice], internal.Mapping) {
	var zero T

	names := [2]string{schema, tableName}
	mappings := internal.GetMappings(reflect.TypeOf(zero))
	allCols := mappings.Columns(names[:]...)

	return &View[T, Tslice]{
		name:    names,
		mapping: mappings,
		allCols: allCols,
	}, mappings
}

type View[T any, Tslice ~[]T] struct {
	name [2]string

	mapping internal.Mapping
	allCols orm.Columns

	AfterSelectHooks orm.Hooks[T]
}

func (v *View[T, Tslice]) Name() Expression {
	return Quote(v.name[:]...)
}

// Returns a column list
func (v *View[T, Tslice]) Columns(ctx context.Context) orm.Columns {
	// get the schema
	return v.allCols
}

// Adds table name et al
func (t *View[T, Tslice]) Query(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := Select(qm.From(t.Name()))
	q.Apply(queryMods...)

	// Append the table columns
	if len(q.Expression.SelectList.Columns) == 0 {
		q.Expression.AppendSelect(t.Columns(ctx))
	}

	return &ViewQuery[T, Tslice]{
		q:                q,
		ctx:              ctx,
		exec:             exec,
		afterSelectHooks: &t.AfterSelectHooks,
	}
}

// Prepare a statement that will be mapped to the view's type
func (v *View[T, Tslice]) Prepare(ctx context.Context, exec bob.Preparer, queryMods ...bob.Mod[*dialect.SelectQuery]) (bob.QueryStmt[T, Tslice], error) {
	return v.PrepareQuery(ctx, exec, v.Query(ctx, nil, queryMods...))
}

// Prepare a statement from an existing query that will be mapped to the view's type
func (v *View[T, Tslice]) PrepareQuery(ctx context.Context, exec bob.Preparer, q bob.Query) (bob.QueryStmt[T, Tslice], error) {
	return bob.PrepareQueryx[T, Tslice](ctx, q, scan.StructMapper[T](), exec, v.afterSelect(ctx, exec))
}

func (v *View[T, Ts]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			for _, val := range retrieved {
				_, err := v.AfterSelectHooks.Do(ctx, exec, val)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}
}

type ViewQuery[T any, Ts ~[]T] struct {
	ctx              context.Context
	exec             bob.Executor
	q                bob.BaseQuery[*dialect.SelectQuery]
	afterSelectHooks *orm.Hooks[T]
}

func (v *ViewQuery[T, Ts]) WriteQuery(w io.Writer, start int) ([]any, error) {
	return v.q.WriteQuery(w, start)
}

func (v *ViewQuery[T, Ts]) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return v.q.WriteSQL(w, d, start)
}

func (v *ViewQuery[T, Ts]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			for _, val := range retrieved {
				_, err := v.afterSelectHooks.Do(ctx, exec, val)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}
}

// First matching row
func (v *ViewQuery[T, Tslice]) One() (T, error) {
	v.q.Expression.Limit.SetLimit(1)
	val, err := bob.One(v.ctx, v.exec, v.q, scan.StructMapper[T](), v.afterSelect(v.ctx, v.exec))
	if err != nil {
		return val, err
	}

	return val, nil
}

// All matching rows
func (v *ViewQuery[T, Tslice]) All() (Tslice, error) {
	vals, err := bob.Allx[T, Tslice](v.ctx, v.exec, v.q, scan.StructMapper[T](), v.afterSelect(v.ctx, v.exec))
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// Cursor to scan through the results
func (v *ViewQuery[T, Tslice]) Cursor() (scan.ICursor[T], error) {
	return bob.Cursor(v.ctx, v.exec, v.q, scan.StructMapper[T](), v.afterSelect(v.ctx, v.exec))
}

// Count the number of matching rows
func (v *ViewQuery[T, Tslice]) Count() (int64, error) {
	v.q.Expression.SelectList.Columns = []any{"count(1)"}
	return bob.One(v.ctx, v.exec, v.q, scan.SingleColumnMapper[int64])
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Tslice]) Exists() (bool, error) {
	count, err := v.Count()
	return count > 0, err
}
