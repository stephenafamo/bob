package mysql

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewView[T any](tableName string) *View[T, []T] {
	return NewViewx[T, []T](tableName)
}

func NewViewx[T any, Tslice ~[]T](tableName string) *View[T, Tslice] {
	v, _ := newView[T, Tslice](tableName)
	return v
}

func newView[T any, Tslice ~[]T](tableName string) (*View[T, Tslice], mappings.Mapping) {
	var zero T

	mappings := mappings.GetMappings(reflect.TypeOf(zero))
	alias := tableName
	allCols := internal.MappingCols(mappings, alias)

	return &View[T, Tslice]{
		name:    tableName,
		alias:   alias,
		allCols: allCols,
		scanner: scan.StructMapper[T](),
	}, mappings
}

type View[T any, Tslice ~[]T] struct {
	name  string
	alias string

	allCols orm.Columns
	scanner scan.Mapper[T]

	AfterSelectHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	SelectQueryHooks bob.Hooks[*dialect.SelectQuery, bob.SkipQueryHooksKey]
}

func (v *View[T, Tslice]) Name() Expression {
	return Quote(v.name)
}

func (v *View[T, Tslice]) NameAs() bob.Expression {
	return v.Name().As(v.alias)
}

func (v *View[T, Tslice]) Alias() string {
	return v.alias
}

// Returns a column list
func (v *View[T, Tslice]) Columns() orm.Columns {
	// get the schema
	return v.allCols
}

// Adds table name et al
func (v *View[T, Tslice]) Query(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := &ViewQuery[T, Tslice]{
		Query: orm.Query[*dialect.SelectQuery, T, Tslice]{
			ExecQuery: orm.ExecQuery[*dialect.SelectQuery]{
				BaseQuery: Select(sm.From(v.NameAs())),
				Hooks:     &v.SelectQueryHooks,
			},
			Scanner: v.scanner,
		},
	}

	q.BaseQuery.Expression.AppendContextualModFunc(
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
