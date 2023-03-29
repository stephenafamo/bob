package commons

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// cannot ergonomically use this until Type parameters are allowed on aliases
// see https://github.com/golang/go/issues/46477

type expression interface {
	As(string) bob.Expression
}

type viewInterface[Q bob.Expression, E any] interface {
	Quote(...string) E
	NewQuery(from bob.Expression) bob.BaseQuery[Q]
}

type queryable interface {
	bob.Expression
	SetSelect(columns ...any)
	SetLimit(any)
	CountSelectCols() int
	AppendSelect(columns ...any)
	SetLoadContext(context.Context)
}

// UseSchema modifies a context to add a schema that will be used when
// a tablle/view was generated with an empty schema
func UseSchema(ctx context.Context, schema string) context.Context {
	return context.WithValue(ctx, orm.CtxUseSchema, schema)
}

func NewView[T any, Ts ~[]T, Q queryable, E expression, I viewInterface[Q, E]](schema, tableName string) (*View[T, Ts, Q, E, I], internal.Mapping) {
	var zero T

	mappings := internal.GetMappings(reflect.TypeOf(zero))
	alias := tableName
	if schema != "" {
		alias = fmt.Sprintf("%s.%s", schema, tableName)
	}

	allCols := mappings.Columns(alias)

	return &View[T, Ts, Q, E, I]{
		schema:  schema,
		name:    tableName,
		alias:   alias,
		mapping: mappings,
		allCols: allCols,
		scanner: scan.StructMapper[T](),
	}, mappings
}

type View[T any, Ts ~[]T, Q queryable, E expression, I viewInterface[Q, E]] struct {
	schema string
	name   string
	alias  string

	mapping internal.Mapping
	allCols orm.Columns
	scanner scan.Mapper[T]

	AfterSelectHooks orm.Hooks[T]

	// the zero value should be good
	creator I
}

func (v *View[T, Ts, Q, E, I]) Name(ctx context.Context) E {
	// schema is not empty, never override
	if v.schema != "" {
		return v.creator.Quote(v.schema, v.name)
	}

	schema, _ := ctx.Value(orm.CtxUseSchema).(string)
	return v.creator.Quote(schema, v.name)
}

func (v *View[T, Ts, Q, E, I]) NameAs(ctx context.Context) bob.Expression {
	return v.Name(ctx).As(v.alias)
}

// Returns a column list
func (v *View[T, Ts, Q, E, I]) Columns() orm.Columns {
	// get the schema
	return v.allCols
}

// Adds table name et al
func (v *View[T, Ts, Q, E, I]) Query(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[Q]) *ViewQuery[T, Ts, Q, E, I] {
	q := &ViewQuery[T, Ts, Q, E, I]{
		BaseQuery: v.creator.NewQuery(v.NameAs(ctx)),
		ctx:       ctx,
		exec:      exec,
		view:      v,
	}

	q.Expression.SetLoadContext(ctx)
	q.Apply(queryMods...)

	return q
}

// Prepare a statement that will be mapped to the view's type
func (v *View[T, Ts, Q, E, I]) Prepare(ctx context.Context, exec bob.Preparer, queryMods ...bob.Mod[Q]) (bob.QueryStmt[T, Ts], error) {
	return v.PrepareQuery(ctx, exec, v.Query(ctx, nil, queryMods...))
}

// Prepare a statement from an existing query that will be mapped to the view's type
func (v *View[T, Ts, Q, E, I]) PrepareQuery(ctx context.Context, exec bob.Preparer, q bob.Query) (bob.QueryStmt[T, Ts], error) {
	return bob.PrepareQueryx[T, Ts](ctx, exec, q, v.scanner, v.afterSelect(ctx, exec))
}

func (v *View[T, Ts, Q, E, I]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
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

type ViewQuery[T any, Ts ~[]T, Q queryable, E expression, I viewInterface[Q, E]] struct {
	bob.BaseQuery[Q]
	ctx  context.Context
	exec bob.Executor
	view *View[T, Ts, Q, E, I]
}

// Satisfies the Expression interface, but uses its own dialect instead
// of the dialect passed to it
// it is necessary to override this method to be able to add columns if not set
func (v ViewQuery[T, Ts, Q, E, I]) WriteSQL(w io.Writer, _ bob.Dialect, start int) ([]any, error) {
	// Append the table columns
	if v.BaseQuery.Expression.CountSelectCols() == 0 {
		v.BaseQuery.Expression.AppendSelect(v.view.Columns())
	}

	return v.Expression.WriteSQL(w, v.Dialect, start)
}

func (v *ViewQuery[T, Ts, Q, E, I]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			for _, val := range retrieved {
				_, err := v.view.AfterSelectHooks.Do(ctx, exec, val)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}
}

// First matching row
func (v *ViewQuery[T, Ts, Q, E, I]) One() (T, error) {
	v.BaseQuery.Expression.SetLimit(1)
	return bob.One(v.ctx, v.exec, v, v.view.scanner, v.afterSelect(v.ctx, v.exec))
}

// All matching rows
func (v *ViewQuery[T, Ts, Q, E, I]) All() (Ts, error) {
	return bob.Allx[T, Ts](v.ctx, v.exec, v, v.view.scanner, v.afterSelect(v.ctx, v.exec))
}

// Cursor to scan through the results
func (v *ViewQuery[T, Ts, Q, E, I]) Cursor() (scan.ICursor[T], error) {
	return bob.Cursor(v.ctx, v.exec, v, v.view.scanner, v.afterSelect(v.ctx, v.exec))
}

// Count the number of matching rows
func (v *ViewQuery[T, Ts, Q, E, I]) Count() (int64, error) {
	v.BaseQuery.Expression.SetSelect("count(1)")
	return bob.One(v.ctx, v.exec, v, scan.SingleColumnMapper[int64])
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Ts, Q, E, I]) Exists() (bool, error) {
	count, err := v.Count()
	return count > 0, err
}
