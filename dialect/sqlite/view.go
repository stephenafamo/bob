package sqlite

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// UseSchema modifies a context to add a schema that will be used when
// a tablle/view was generated with an empty schema
func UseSchema(ctx context.Context, schema string) context.Context {
	return context.WithValue(ctx, orm.CtxUseSchema, schema)
}

func NewView[T any](schema, tableName string) *View[T, []T] {
	return NewViewx[T, []T](schema, tableName)
}

func NewViewx[T any, Tslice ~[]T](schema, tableName string) *View[T, Tslice] {
	v, _ := newView[T, Tslice](schema, tableName)
	return v
}

func newView[T any, Tslice ~[]T](schema, tableName string) (*View[T, Tslice], internal.Mapping) {
	var zero T

	mappings := internal.GetMappings(reflect.TypeOf(zero))
	alias := tableName
	if schema != "" {
		alias = fmt.Sprintf("%s.%s", schema, tableName)
	}

	allCols := mappings.Columns(alias)

	return &View[T, Tslice]{
		schema:  schema,
		name:    tableName,
		alias:   alias,
		mapping: mappings,
		allCols: allCols,
		scanner: scan.StructMapper[T](),
	}, mappings
}

type View[T any, Tslice ~[]T] struct {
	schema string
	name   string
	alias  string

	mapping internal.Mapping
	allCols orm.Columns
	scanner scan.Mapper[T]

	AfterSelectHooks orm.Hooks[Tslice]
}

func (v *View[T, Tslice]) Name(ctx context.Context) Expression {
	// schema is not empty, never override
	if v.schema != "" {
		return Quote(v.schema, v.name)
	}

	schema, _ := ctx.Value(orm.CtxUseSchema).(string)
	return Quote(schema, v.name)
}

func (v *View[T, Tslice]) NameAs(ctx context.Context) bob.Expression {
	return v.Name(ctx).As(v.alias)
}

// Returns a column list
func (v *View[T, Tslice]) Columns() orm.Columns {
	// get the schema
	return v.allCols
}

// Adds table name et al
func (v *View[T, Tslice]) Query(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := &ViewQuery[T, Tslice]{
		BaseQuery: Select(sm.From(v.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      v,
	}

	q.Expression.SetLoadContext(ctx)
	q.Apply(queryMods...)

	return q
}

// Prepare a statement that will be mapped to the view's type
func (v *View[T, Tslice]) Prepare(ctx context.Context, exec bob.Preparer, queryMods ...bob.Mod[*dialect.SelectQuery]) (bob.QueryStmt[T, Tslice], error) {
	return v.PrepareQuery(ctx, exec, v.Query(ctx, nil, queryMods...))
}

// Prepare a statement from an existing query that will be mapped to the view's type
func (v *View[T, Tslice]) PrepareQuery(ctx context.Context, exec bob.Preparer, q bob.Query) (bob.QueryStmt[T, Tslice], error) {
	return bob.PrepareQueryx[T, Tslice](ctx, exec, q, scan.StructMapper[T](), v.afterSelect(ctx, exec))
}

func (v *View[T, Ts]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			_, err := v.AfterSelectHooks.Do(ctx, exec, retrieved)
			if err != nil {
				return err
			}

			return nil
		}
	}
}

type ViewQuery[T any, Ts ~[]T] struct {
	bob.BaseQuery[*dialect.SelectQuery]
	ctx  context.Context
	exec bob.Executor
	view *View[T, Ts]
}

// it is necessary to override this method to be able to add columns if not set
func (v ViewQuery[T, Ts]) WriteSQL(w io.Writer, _ bob.Dialect, start int) ([]any, error) {
	// Append the table columns
	if len(v.BaseQuery.Expression.SelectList.Columns) == 0 {
		v.BaseQuery.Expression.AppendSelect(v.view.Columns())
	}

	return v.Expression.WriteSQL(w, v.Dialect, start)
}

// it is necessary to override this method to be able to add columns if not set
func (v ViewQuery[T, Ts]) WriteQuery(w io.Writer, start int) ([]any, error) {
	// Append the table columns
	if len(v.BaseQuery.Expression.SelectList.Columns) == 0 {
		v.BaseQuery.Expression.AppendSelect(v.view.Columns())
	}

	return v.BaseQuery.WriteQuery(w, start)
}

func (v *ViewQuery[T, Ts]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			_, err := v.view.AfterSelectHooks.Do(ctx, exec, retrieved)
			if err != nil {
				return err
			}

			return nil
		}
	}
}

// First matching row
func (v *ViewQuery[T, Tslice]) One() (T, error) {
	v.BaseQuery.Expression.Limit.SetLimit(1)
	return bob.One(v.ctx, v.exec, v, v.view.scanner, v.afterSelect(v.ctx, v.exec))
}

// All matching rows
func (v *ViewQuery[T, Tslice]) All() (Tslice, error) {
	return bob.Allx[T, Tslice](v.ctx, v.exec, v, v.view.scanner, v.afterSelect(v.ctx, v.exec))
}

// Cursor to scan through the results
func (v *ViewQuery[T, Tslice]) Cursor() (scan.ICursor[T], error) {
	return bob.Cursor(v.ctx, v.exec, v, v.view.scanner, v.afterSelect(v.ctx, v.exec))
}

// Count the number of matching rows
func (v *ViewQuery[T, Tslice]) Count() (int64, error) {
	v.BaseQuery.Expression.SelectList.Columns = []any{"count(1)"}
	return bob.One(v.ctx, v.exec, v, scan.SingleColumnMapper[int64])
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Tslice]) Exists() (bool, error) {
	count, err := v.Count()
	return count > 0, err
}
