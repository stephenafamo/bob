package psql

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// UseSchema modifies a context to add a schema that will be used when
// a tablle/view was generated with an empty schema
func UseSchema(ctx context.Context, schema string) context.Context {
	return context.WithValue(ctx, orm.CtxUseSchema, schema)
}

func NewView[T any, C bob.Expression](schema, tableName string, columns C) *View[T, []T, C] {
	return NewViewx[T, []T](schema, tableName, columns)
}

func NewViewx[T any, Tslice ~[]T, C bob.Expression](schema, tableName string, columns C) *View[T, Tslice, C] {
	v, _ := newView[T, Tslice](schema, tableName, columns)
	return v
}

func newView[T any, Tslice ~[]T, C bob.Expression](schema, tableName string, columns C) (*View[T, Tslice, C], mappings.Mapping) {
	mappings := mappings.GetMappings(reflect.TypeOf(*new(T)))
	alias := tableName
	if schema != "" {
		alias = fmt.Sprintf("%s.%s", schema, tableName)
	}

	return &View[T, Tslice, C]{
		schema:  schema,
		name:    tableName,
		alias:   alias,
		allCols: expr.NewColumnsExpr(mappings.All...).WithParent(alias),
		scanner: scan.StructMapper[T](),
		Columns: columns,
	}, mappings
}

type View[T any, Tslice ~[]T, C bob.Expression] struct {
	schema string
	name   string
	alias  string

	allCols expr.ColumnsExpr
	scanner scan.Mapper[T]

	Columns C

	AfterSelectHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	SelectQueryHooks bob.Hooks[*dialect.SelectQuery, bob.SkipQueryHooksKey]
}

// NameExpr returns the table name as an expression
func (v *View[T, Tslice, C]) NameExpr() Expression {
	// schema is not empty, never override
	if v.schema != "" {
		return Quote(v.schema, v.name)
	}

	return Expression{}.New(orm.SchemaTable(v.name))
}

// NameAsExpr returns the table name as an expression with an alias
func (v *View[T, Tslice, C]) NameAsExpr() bob.Expression {
	return v.NameExpr().As(v.alias)
}

// Alias returns the alias
func (v *View[T, Tslice, C]) Alias() string {
	return v.alias
}

// Schema returns the schema name for the view
func (v *View[T, Tslice, C]) Schema() string {
	return v.schema
}

// Name returns the view (table/view) name
func (v *View[T, Tslice, C]) Name() string {
	return v.name
}

// ColumnsExpr returns a column list expression
func (v *View[T, Tslice, C]) ColumnsExpr() expr.ColumnsExpr {
	// get the schema
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

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.SelectQuery) (context.Context, error) {
			if len(q.SelectList.Columns) == 0 {
				q.AppendSelect(v.Columns)
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
