package mysql

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewView[T any, C bob.Expression](tableName string, columns C) *View[T, []T, C] {
	return NewViewx[T, []T](tableName, columns, nil)
}

// NewViewx creates a new View with a custom scanner.
// If scanner is nil, it falls back to [scan.StructMapper].
func NewViewx[T any, Tslice ~[]T, C bob.Expression](tableName string, columns C, scanner scan.Mapper[T]) *View[T, Tslice, C] {
	v, _ := newView[T, Tslice](tableName, columns, scanner)
	return v
}

func newView[T any, Tslice ~[]T, C bob.Expression](tableName string, columns C, scanner scan.Mapper[T]) (*View[T, Tslice, C], mappings.Mapping) {
	if scanner == nil {
		scanner = scan.StructMapper[T]()
	}

	mappings := mappings.GetMappings(reflect.TypeOf(*new(T)))

	return &View[T, Tslice, C]{
		name:    tableName,
		alias:   tableName,
		allCols: expr.NewColumnsExpr(mappings.All...).WithParent(tableName),
		scanner: scanner,
		Columns: columns,
	}, mappings
}

type View[T any, Tslice ~[]T, C bob.Expression] struct {
	name  string
	alias string

	allCols expr.ColumnsExpr
	scanner scan.Mapper[T]

	Columns C

	AfterSelectHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	SelectQueryHooks bob.Hooks[*dialect.SelectQuery, bob.SkipQueryHooksKey]
}

// NameExpr returns the table name as an expression
func (v *View[T, Tslice, C]) NameExpr() Expression {
	return Quote(v.name)
}

// NameAsExpr returns the table name as an expression with an alias when needed.
func (v *View[T, Tslice, C]) NameAsExpr() bob.Expression {
	expr := v.NameExpr()
	if v.alias != v.name {
		return expr.As(v.alias)
	}
	return expr
}

// Alias returns the alias
func (v *View[T, Tslice, C]) Alias() string {
	return v.alias
}

// Name returns the view (table/view) name
func (v *View[T, Tslice, C]) Name() string {
	return v.name
}

// ColumnsExpr returns a column list expression
func (v *View[T, Tslice, C]) ColumnsExpr() expr.ColumnsExpr {
	return v.allCols
}

// Query starts a select query on the view
func (v *View[T, Tslice, C]) Query(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := &ViewQuery[T, Tslice]{
		Query: orm.Query[*dialect.SelectQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]{
			ExecQuery: orm.ExecQuery[*dialect.SelectQuery]{
				BaseQuery: Select(sm.From(v.NameAsExpr())),
				Hooks:     &v.SelectQueryHooks,
			},
			Scanner: v.scanner,
		},
	}

	q.BaseQuery.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.SelectQuery) (context.Context, error) {
			if len(q.SelectList.Columns) == 0 {
				q.AppendSelect(v.ColumnsExpr())
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

type ViewQuery[T any, Ts ~[]T] struct {
	orm.Query[*dialect.SelectQuery, T, Ts, bob.SliceTransformer[T, Ts]]
}

func (v *ViewQuery[T, Ts]) Clone() *ViewQuery[T, Ts] {
	if v == nil {
		return nil
	}

	return &ViewQuery[T, Ts]{
		Query: v.Query.Clone(),
	}
}

func (v *ViewQuery[T, Ts]) With(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Ts] {
	clone := v.Clone()
	clone.Apply(queryMods...)

	return clone
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
	countQuery.Expression.ClearOrderBy()
	// remove group by
	countQuery.Expression.SetGroups()
	// remove offset
	countQuery.Expression.SetOffset(0)

	return countQuery
}
