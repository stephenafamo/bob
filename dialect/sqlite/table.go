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
	setter[T any]                      = orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]
	ormInsertQuery[T any, Tslice ~[]T] = orm.Query[*dialect.InsertQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormUpdateQuery[T any, Tslice ~[]T] = orm.Query[*dialect.UpdateQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormDeleteQuery[T any, Tslice ~[]T] = orm.Query[*dialect.DeleteQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
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

	InsertQueryHooks bob.Hooks[*dialect.InsertQuery, bob.SkipQueryHooksKey]
	UpdateQueryHooks bob.Hooks[*dialect.UpdateQuery, bob.SkipQueryHooksKey]
	DeleteQueryHooks bob.Hooks[*dialect.DeleteQuery, bob.SkipQueryHooksKey]
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset, C]) Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) *ormInsertQuery[T, Tslice] {
	q := &ormInsertQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.InsertQuery]{
			BaseQuery: Insert(im.Into(t.NameAsExpr())),
			Hooks:     &t.InsertQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.InsertQuery) (context.Context, error) {
			if !q.HasReturning() {
				q.AppendReturning(t.returningCols)
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

// Starts an Update query for this table
func (t *Table[T, Tslice, Tset, C]) Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) *ormUpdateQuery[T, Tslice] {
	q := &ormUpdateQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.UpdateQuery]{
			BaseQuery: Update(um.Table(t.NameAsExpr())),
			Hooks:     &t.UpdateQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.UpdateQuery) (context.Context, error) {
			if !q.HasReturning() {
				q.AppendReturning(t.returningCols)
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

// Starts a Delete query for this table
func (t *Table[T, Tslice, Tset, C]) Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) *ormDeleteQuery[T, Tslice] {
	q := &ormDeleteQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.DeleteQuery]{
			BaseQuery: Delete(dm.From(t.NameAsExpr())),
			Hooks:     &t.DeleteQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.DeleteQuery) (context.Context, error) {
			if !q.HasReturning() {
				q.AppendReturning(t.returningCols)
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}
