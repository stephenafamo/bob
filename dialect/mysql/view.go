package mysql

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/internal/ormruntime"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewView[T any, C bob.Expression](tableName string, columns C) *View[T, []T, C] {
	return NewViewx[T, []T](tableName, columns, nil)
}

// NewViewx creates a new View with a custom scanner.
// If scanner is nil, it falls back to [scan.StructMapper].
func NewViewx[T any, Tslice ~[]T, C bob.Expression](tableName string, columns C, scanner scan.Mapper[T]) *View[T, Tslice, C] {
	allCols := mappings.SQLColumns[T](columns, tableName, scanner != nil)
	projection := mappings.Projection(columns, allCols, scanner != nil)
	if scanner == nil {
		scanner = scan.StructMapper[T]()
	}

	return &View[T, Tslice, C]{
		name:       tableName,
		alias:      tableName,
		allCols:    allCols,
		projection: projection,
		scanner:    scanner,
		Columns:    columns,
	}
}

type View[T any, Tslice ~[]T, C bob.Expression] struct {
	name  string
	alias string

	allCols    expr.ColumnsExpr
	projection bob.Expression
	scanner    scan.Mapper[T]

	Columns C

	AfterSelectHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	SelectQueryHooks bob.QueryHooks[SelectEditor]
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
func (v *View[T, Tslice, C]) Query(queryMods ...bob.Mod[dialect.SelectEditor]) *ViewQuery[T, Tslice] {
	q := dialect.NewSelect(v.SelectQueryHooks.GetHooks(), bob.ModFunc[dialect.SelectEditor](func(ast dialect.SelectEditor) {
		sm.From(v.NameAsExpr()).Edit(ast)
		ast.AppendContextualModFunc(func(ctx context.Context, q dialect.SelectEditor) (context.Context, error) {
			if q.CountSelectCols() == 0 {
				q.AppendSelect(v.projection)
			}
			return ctx, nil
		})
		bob.Mods[dialect.SelectEditor](queryMods).Edit(ast)
	}))
	return &ViewQuery[T, Tslice]{*ormruntime.ModelQuery[Tslice](q, v.scanner)}
}

type ViewQuery[T any, Ts ~[]T] struct {
	orm.Query[dialect.SelectEditor, T, Ts, bob.SliceTransformer[T, Ts]]
}

func (v ViewQuery[T, Ts]) WithMods(mods ...bob.Mod[dialect.SelectEditor]) *ViewQuery[T, Ts] {
	v.QueryBuilder = v.QueryBuilder.WithMods(mods...)
	return &v
}

func (v *ViewQuery[T, Tslice]) Count(ctx context.Context, exec bob.Executor) (int64, error) {
	return ormruntime.Count(ctx, exec, v.ExecQuery)
}

// Exists checks if there is any matching row
func (v *ViewQuery[T, Tslice]) Exists(ctx context.Context, exec bob.Executor) (bool, error) {
	count, err := v.Count(ctx, exec)
	return count > 0, err
}
