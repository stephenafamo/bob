package psql

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
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

func newView[T any, Tslice ~[]T](schema, tableName string) (*View[T, Tslice], mappings.Mapping) {
	var zero T

	mappings := mappings.GetMappings(reflect.TypeOf(zero))
	alias := tableName
	if schema != "" {
		alias = fmt.Sprintf("%s.%s", schema, tableName)
	}

	allCols := internal.MappingCols(mappings, alias)

	return &View[T, Tslice]{
		schema:  schema,
		name:    tableName,
		alias:   alias,
		allCols: allCols,
		scanner: scan.StructMapper[T](),
	}, mappings
}

type View[T any, Tslice ~[]T] struct {
	schema string
	name   string
	alias  string

	allCols orm.Columns
	scanner scan.Mapper[T]

	AfterSelectHooks orm.Hooks[Tslice, orm.SkipModelHooksKey]
	SelectQueryHooks orm.Hooks[*dialect.SelectQuery, orm.SkipQueryHooksKey]
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

// Starts a select query
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
	return bob.PrepareQueryx[T, Tslice](ctx, exec, q, v.scanner, v.afterSelect(exec))
}

func (v *View[T, Ts]) afterSelect(exec bob.Executor) bob.ExecOption[T] {
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

	return v.BaseQuery.WriteSQL(w, v.Dialect, start)
}

// it is necessary to override this method to be able to add columns if not set
func (v ViewQuery[T, Ts]) WriteQuery(w io.Writer, start int) ([]any, error) {
	// Append the table columns
	if len(v.BaseQuery.Expression.SelectList.Columns) == 0 {
		v.BaseQuery.Expression.AppendSelect(v.view.Columns())
	}

	return v.BaseQuery.WriteQuery(w, start)
}

func (v *ViewQuery[T, Ts]) hook() error {
	var err error
	v.ctx, err = v.view.SelectQueryHooks.Do(v.ctx, v.exec, v.Expression)
	return err
}

// Execute the query
func (v *ViewQuery[T, Tslice]) Exec() (int64, error) {
	if err := v.hook(); err != nil {
		return 0, err
	}

	result, err := v.BaseQuery.Exec(v.ctx, v.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// First matching row
func (v *ViewQuery[T, Tslice]) One() (T, error) {
	v.BaseQuery.Expression.Limit.SetLimit(1)
	if err := v.hook(); err != nil {
		return *new(T), err
	}
	return bob.One(v.ctx, v.exec, v, v.view.scanner, v.view.afterSelect(v.exec))
}

// All matching rows
func (v *ViewQuery[T, Tslice]) All() (Tslice, error) {
	if err := v.hook(); err != nil {
		return nil, err
	}
	return bob.Allx[T, Tslice](v.ctx, v.exec, v, v.view.scanner, v.view.afterSelect(v.exec))
}

// Cursor to scan through the results
func (v *ViewQuery[T, Tslice]) Cursor() (scan.ICursor[T], error) {
	if err := v.hook(); err != nil {
		return nil, err
	}
	return bob.Cursor(v.ctx, v.exec, v, v.view.scanner, v.view.afterSelect(v.exec))
}

// Count the number of matching rows
func (v *ViewQuery[T, Tslice]) Count() (int64, error) {
	if err := v.hook(); err != nil {
		return 0, err
	}
	return bob.One(v.ctx, v.exec, asCountQuery(v.BaseQuery), scan.SingleColumnMapper[int64])
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Tslice]) Exists() (bool, error) {
	count, err := v.Count()
	return count > 0, err
}

// asCountQuery clones and rewrites an existing query to a count query
func asCountQuery(query bob.BaseQuery[*dialect.SelectQuery]) bob.BaseQuery[*dialect.SelectQuery] {
	// clone the original query, so it's not being modified silently
	countQuery := query.Clone()
	// only select the count
	countQuery.Expression.SetSelect("count(1)")
	// don't select any preload columns
	countQuery.Expression.SetPreloadSelect()
	// disable mapper mods
	countQuery.Expression.SetMapperMods()
	// disable loaders
	countQuery.Expression.SetLoaders()
	// set the limit to 1
	countQuery.Expression.SetLimit(1)
	// remove ordering
	countQuery.Expression.SetOrderBy()
	// remove group by
	countQuery.Expression.SetGroups()
	// remove offset
	countQuery.Expression.SetOffset(0)

	return countQuery
}
