package sqlite

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/dm"
	"github.com/stephenafamo/bob/dialect/sqlite/im"
	"github.com/stephenafamo/bob/dialect/sqlite/um"
	"github.com/stephenafamo/bob/internal/ormruntime"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type (
	rowOperations[T any, S ~[]T]       = ormruntime.Rows[T, S, *ViewQuery[T, S]]
	setter[T any]                      = orm.Setter[T, dialect.InsertEditor, dialect.UpdateEditor]
	ormInsertQuery[T any, Tslice ~[]T] = orm.Query[dialect.InsertEditor, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormUpdateQuery[T any, Tslice ~[]T] = orm.Query[dialect.UpdateEditor, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormDeleteQuery[T any, Tslice ~[]T] = orm.Query[dialect.DeleteEditor, T, Tslice, bob.SliceTransformer[T, Tslice]]
)

func NewTable[T any, Tset setter[T], C bob.Expression](
	schema, tableName string, columns C, rows orm.RowMapping[T, []T],
) *Table[T, []T, Tset, C] {
	return NewTablex[T, []T, Tset](schema, tableName, columns, nil, rows)
}

// NewTablex creates a Table with a row mapping and a custom scanner.
// If scanner is nil, it falls back to [scan.StructMapper].
func NewTablex[T any, Tslice ~[]T, Tset setter[T], C bob.Expression](
	schema, tableName string,
	columns C,
	scanner scan.Mapper[T],
	rows orm.RowMapping[T, Tslice],
) *Table[T, Tslice, Tset, C] {
	view := NewViewx[T, Tslice](schema, tableName, columns, scanner)
	return &Table[T, Tslice, Tset, C]{
		View:          view,
		rowOperations: ormruntime.NewRows(rows, view.lookup),
	}
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset setter[T], C bob.Expression] struct {
	*View[T, Tslice, C]
	rowOperations[T, Tslice]

	BeforeInsertHooks bob.Hooks[Tset, bob.SkipModelHooksKey]
	AfterInsertHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeUpdateHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterUpdateHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeDeleteHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterDeleteHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	InsertQueryHooks bob.QueryHooks[InsertEditor]
	UpdateQueryHooks bob.QueryHooks[UpdateEditor]
	DeleteQueryHooks bob.QueryHooks[DeleteEditor]
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset, C]) Insert(queryMods ...bob.Mod[dialect.InsertEditor]) *ormInsertQuery[T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewInsert(t.InsertQueryHooks.GetHooks(), bob.ModFunc[dialect.InsertEditor](func(ast dialect.InsertEditor) {
		im.Into(t.NameAsExpr()).Edit(ast)
		ast.AppendContextualModFunc(
			func(ctx context.Context, q dialect.InsertEditor) (context.Context, error) {
				if !q.HasReturning() {
					q.AppendReturning(t.returningCols)
				}
				return ctx, nil
			},
		)
		bob.Mods[dialect.InsertEditor](queryMods).Edit(ast)
	})), t.scanner)
}

// Starts an Update query for this table
func (t *Table[T, Tslice, Tset, C]) Update(queryMods ...bob.Mod[dialect.UpdateEditor]) *ormUpdateQuery[T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewUpdate(t.UpdateQueryHooks.GetHooks(), bob.ModFunc[dialect.UpdateEditor](func(ast dialect.UpdateEditor) {
		um.Table(t.NameAsExpr()).Edit(ast)
		ast.AppendContextualModFunc(
			func(ctx context.Context, q dialect.UpdateEditor) (context.Context, error) {
				if !q.HasReturning() {
					q.AppendReturning(t.returningCols)
				}
				return ctx, nil
			},
		)
		bob.Mods[dialect.UpdateEditor](queryMods).Edit(ast)
	})), t.scanner)
}

// Starts a Delete query for this table
func (t *Table[T, Tslice, Tset, C]) Delete(queryMods ...bob.Mod[dialect.DeleteEditor]) *ormDeleteQuery[T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewDelete(t.DeleteQueryHooks.GetHooks(), bob.ModFunc[dialect.DeleteEditor](func(ast dialect.DeleteEditor) {
		dm.From(t.NameAsExpr()).Edit(ast)
		ast.AppendContextualModFunc(
			func(ctx context.Context, q dialect.DeleteEditor) (context.Context, error) {
				if !q.HasReturning() {
					q.AppendReturning(t.returningCols)
				}
				return ctx, nil
			},
		)
		bob.Mods[dialect.DeleteEditor](queryMods).Edit(ast)
	})), t.scanner)
}
