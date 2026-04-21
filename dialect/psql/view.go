package psql

import (
	"context"
	"fmt"
	"io"
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

func (v *View[T, Tslice, C]) Name() Expression {
	// schema is not empty, never override
	if v.schema != "" {
		return Quote(v.schema, v.name)
	}

	return Expression{}.New(orm.SchemaTable(v.name))
}

func (v *View[T, Tslice, C]) NameAs() bob.Expression {
	return v.Name().As(v.alias)
}

func (v *View[T, Tslice, C]) Alias() string {
	return v.alias
}

// Starts a select query
func (v *View[T, Tslice, C]) Query(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Tslice] {
	q := &ViewQuery[T, Tslice]{
		Query:   Select(sm.From(v.NameAs())),
		Scanner: v.scanner,
		Hooks:   &v.SelectQueryHooks,
	}
	q.Query.derivedSelectQuery.state.DefaultSelectColumns = []any{v.Columns}
	if len(queryMods) == 0 {
		return q
	}
	return q.Apply(queryMods...)
}

type ViewQuery[T any, Ts ~[]T] struct {
	Query   SelectQuery
	Scanner scan.Mapper[T]
	Hooks   *bob.Hooks[*dialect.SelectQuery, bob.SkipQueryHooksKey]
}

func (q *ViewQuery[T, Ts]) With(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Ts] {
	next := *q
	next.Query = next.Query.Apply(queryMods...)
	return &next
}

func (q *ViewQuery[T, Ts]) Apply(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Ts] {
	return q.With(queryMods...)
}

func (q *ViewQuery[T, Ts]) Type() bob.QueryType {
	return q.Query.Type()
}

func (q *ViewQuery[T, Ts]) Build(ctx context.Context) (string, []any, error) {
	return q.Query.Build(ctx)
}

func (q *ViewQuery[T, Ts]) BuildN(ctx context.Context, start int) (string, []any, error) {
	return q.Query.BuildN(ctx, start)
}

func (q *ViewQuery[T, Ts]) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	return q.Query.WriteQuery(ctx, w, start)
}

func (q *ViewQuery[T, Ts]) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return q.Query.WriteSQL(ctx, w, d, start)
}

// Count the number of matching rows
func (v *ViewQuery[T, Tslice]) Count(ctx context.Context, exec bob.Executor) (int64, error) {
	ctx, err := v.RunHooks(ctx, exec)
	if err != nil {
		return 0, err
	}
	sql, args, err := v.Query.AsCount().Build(ctx)
	if err != nil {
		return 0, err
	}
	return scan.One(ctx, exec, scan.SingleColumnMapper[int64], sql, args...)
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Tslice]) Exists(ctx context.Context, exec bob.Executor) (bool, error) {
	count, err := v.Count(ctx, exec)
	return count > 0, err
}

func (q *ViewQuery[T, Ts]) One(ctx context.Context, exec bob.Executor) (T, error) {
	return bob.One(ctx, exec, q, q.Scanner)
}

func (q *ViewQuery[T, Ts]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	return bob.Allx[bob.SliceTransformer[T, Ts]](ctx, exec, q, q.Scanner)
}

func (q *ViewQuery[T, Ts]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	return bob.Cursor(ctx, exec, q, q.Scanner)
}

func (q *ViewQuery[T, Ts]) Each(ctx context.Context, exec bob.Executor) (func(func(T, error) bool), error) {
	return bob.Each(ctx, exec, q, q.Scanner)
}

func (q *ViewQuery[T, Ts]) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	ctx, err := q.Query.RunHooks(ctx, exec)
	if err != nil {
		return ctx, err
	}

	if q.Hooks == nil {
		return ctx, nil
	}

	return q.Hooks.RunHooks(ctx, exec, q.Query.baseQuery().Expression)
}

func (q *ViewQuery[T, Ts]) GetLoaders() []bob.Loader {
	return q.Query.GetLoaders()
}

func (q *ViewQuery[T, Ts]) GetMapperMods() []scan.MapperMod {
	return q.Query.GetMapperMods()
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
