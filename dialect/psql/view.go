package psql

import (
	"context"
	"fmt"
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

	AfterSelectHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	SelectQueryHooks bob.Hooks[*dialect.SelectQuery, bob.SkipQueryHooksKey]
}

func (v *View[T, Tslice]) Name() Expression {
	// schema is not empty, never override
	if v.schema != "" {
		return Quote(v.schema, v.name)
	}

	return Expression{}.New(orm.SchemaTable(v.name))
}

func (v *View[T, Tslice]) NameAs() bob.Expression {
	return v.Name().As(v.alias)
}

// Returns a column list
func (v *View[T, Tslice]) Columns() orm.Columns {
	// get the schema
	return v.allCols
}

// Starts a select query
func (v *View[T, Tslice]) Query(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := &ViewQuery[T, Tslice]{
		Query: orm.Query[*dialect.SelectQuery, T, Tslice]{
			ExecQuery: orm.ExecQuery[*dialect.SelectQuery, T, Tslice]{
				BaseQuery: Select(sm.From(v.NameAs())),
				Scanner:   v.scanner,
				Hooks:     &v.SelectQueryHooks,
			},
		},
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.SelectQuery) (context.Context, error) {
			if len(q.SelectList.Columns) == 0 {
				q.AppendSelect(v.Columns())
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

type ViewQuery[T any, Ts ~[]T] struct {
	orm.Query[*dialect.SelectQuery, T, Ts]
}

// Count the number of matching rows
func (v *ViewQuery[T, Tslice]) Count(ctx context.Context, exec bob.Executor) (int64, error) {
	ctx, err := v.RunHooks(ctx, exec)
	if err != nil {
		return 0, err
	}
	return bob.One(ctx, exec, asCountQuery(v.BaseQuery), scan.SingleColumnMapper[int64])
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Tslice]) Exists(ctx context.Context, exec bob.Executor) (bool, error) {
	count, err := v.Count(ctx, exec)
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
