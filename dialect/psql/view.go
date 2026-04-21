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
		SelectQuery: Select(sm.From(v.NameAs())),
		Scanner:     v.scanner,
		Hooks:       &v.SelectQueryHooks,
	}

	q.SelectQuery.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.SelectQuery) (context.Context, error) {
			if len(q.SelectList.Columns) == 0 {
				q.AppendSelect(v.Columns)
			}
			return ctx, nil
		},
	)

	return q.Apply(queryMods...)
}

type ViewQuery[T any, Ts ~[]T] struct {
	SelectQuery
	Scanner scan.Mapper[T]
	Hooks   *bob.Hooks[*dialect.SelectQuery, bob.SkipQueryHooksKey]
}

func (q *ViewQuery[T, Ts]) With(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Ts] {
	if q == nil {
		return nil
	}

	next := *q
	next.SelectQuery = next.SelectQuery.Apply(queryMods...)
	return &next
}

func (q *ViewQuery[T, Ts]) Apply(queryMods ...bob.Mod[*dialect.SelectQuery]) *ViewQuery[T, Ts] {
	return q.With(queryMods...)
}

// Count the number of matching rows
func (q *ViewQuery[T, Tslice]) Count(ctx context.Context, exec bob.Executor) (int64, error) {
	ctx, err := q.RunHooks(ctx, exec)
	if err != nil {
		return 0, err
	}
	sql, args, err := q.AsCount().Build(ctx)
	if err != nil {
		return 0, err
	}
	return scan.One(ctx, exec, scan.SingleColumnMapper[int64], sql, args...)
}

// Exists checks if there is any matching row
func (q *ViewQuery[T, Tslice]) Exists(ctx context.Context, exec bob.Executor) (bool, error) {
	count, err := q.Count(ctx, exec)
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
	ctx, err := q.SelectQuery.RunHooks(ctx, exec)
	if err != nil {
		return ctx, err
	}

	if q.Hooks == nil {
		return ctx, nil
	}

	return q.Hooks.RunHooks(ctx, exec, q.SelectQuery.Expression)
}
