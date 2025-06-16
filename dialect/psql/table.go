package psql

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
)

type (
	setter[T any]                      = orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]
	ormInsertQuery[T any, Tslice ~[]T] = orm.Query[*dialect.InsertQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormUpdateQuery[T any, Tslice ~[]T] = orm.Query[*dialect.UpdateQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormDeleteQuery[T any, Tslice ~[]T] = orm.Query[*dialect.DeleteQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
)

func NewTable[T any, Tset setter[T]](schema, tableName string) *Table[T, []T, Tset] {
	return NewTablex[T, []T, Tset](schema, tableName)
}

func NewTablex[T any, Tslice ~[]T, Tset setter[T]](schema, table string) *Table[T, Tslice, Tset] {
	setMapping := mappings.GetMappings(reflect.TypeOf((*new(Tset))))
	view, mappings := newView[T, Tslice](schema, table)
	t := &Table[T, Tslice, Tset]{
		View:             view,
		pkCols:           orm.NewColumns(mappings.PKs...).WithParent(view.alias),
		setterMapping:    setMapping,
		nonGeneratedCols: internal.FilterNonZero(mappings.NonGenerated),
	}

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset setter[T]] struct {
	*View[T, Tslice]
	pkCols           orm.Columns
	setterMapping    mappings.Mapping
	nonGeneratedCols []string

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

// Returns the primary key columns for this table.
func (t *Table[T, Tslice, Tset]) PrimaryKey() orm.Columns {
	return t.pkCols
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset]) Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) *ormInsertQuery[T, Tslice] {
	q := &ormInsertQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.InsertQuery]{
			BaseQuery: Insert(im.Into(t.NameAs(), t.nonGeneratedCols...)),
			Hooks:     &t.InsertQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.InsertQuery) (context.Context, error) {
			if !q.HasReturning() {
				q.AppendReturning(t.Columns())
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

// Starts an Update query for this table
func (t *Table[T, Tslice, Tset]) Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) *ormUpdateQuery[T, Tslice] {
	q := &ormUpdateQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.UpdateQuery]{
			BaseQuery: Update(um.Table(t.NameAs())),
			Hooks:     &t.UpdateQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.UpdateQuery) (context.Context, error) {
			if !q.HasReturning() {
				q.AppendReturning(t.Columns())
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

// Starts a Delete query for this table
func (t *Table[T, Tslice, Tset]) Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) *ormDeleteQuery[T, Tslice] {
	q := &ormDeleteQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.DeleteQuery]{
			BaseQuery: Delete(dm.From(t.NameAs())),
			Hooks:     &t.DeleteQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.DeleteQuery) (context.Context, error) {
			if !q.HasReturning() {
				q.AppendReturning(t.Columns())
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}
