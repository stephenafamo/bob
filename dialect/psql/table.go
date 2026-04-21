package psql

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/mm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	bobmods "github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/orm"
)

type (
	setter[T any]                      = orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]
	ormInsertQuery[T any, Tslice ~[]T] = orm.Query[*dialect.InsertQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormUpdateQuery[T any, Tslice ~[]T] = orm.Query[*dialect.UpdateQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormDeleteQuery[T any, Tslice ~[]T] = orm.Query[*dialect.DeleteQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	ormMergeQuery[T any, Tslice ~[]T]  = orm.Query[*dialect.MergeQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
)

func NewTable[T any, Tset setter[T], C bob.Expression](schema, tableName string, columns C) *Table[T, []T, Tset, C] {
	return NewTablex[T, []T, Tset](schema, tableName, columns)
}

func NewTablex[T any, Tslice ~[]T, Tset setter[T], C bob.Expression](schema, table string, columns C) *Table[T, Tslice, Tset, C] {
	setMapping := mappings.GetMappings(reflect.TypeOf((*new(Tset))))
	view, mappings := newView[T, Tslice](schema, table, columns)
	t := &Table[T, Tslice, Tset, C]{
		View:             view,
		pkCols:           expr.NewColumnsExpr(mappings.PKs...).WithParent(view.alias),
		setterMapping:    setMapping,
		nonGeneratedCols: internal.FilterNonZero(mappings.NonGenerated),
	}

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset setter[T], C bob.Expression] struct {
	*View[T, Tslice, C]
	pkCols           expr.ColumnsExpr
	setterMapping    mappings.Mapping
	nonGeneratedCols []string

	BeforeInsertHooks bob.Hooks[Tset, bob.SkipModelHooksKey]
	AfterInsertHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeUpdateHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterUpdateHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeDeleteHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterDeleteHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeMergeHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterMergeHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	InsertQueryHooks bob.Hooks[*dialect.InsertQuery, bob.SkipQueryHooksKey]
	UpdateQueryHooks bob.Hooks[*dialect.UpdateQuery, bob.SkipQueryHooksKey]
	DeleteQueryHooks bob.Hooks[*dialect.DeleteQuery, bob.SkipQueryHooksKey]
	MergeQueryHooks  bob.Hooks[*dialect.MergeQuery, bob.SkipQueryHooksKey]
}

// Returns the primary key columns for this table.
func (t *Table[T, Tslice, Tset, C]) PrimaryKey() expr.ColumnsExpr {
	return t.pkCols
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset, C]) Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) *ormInsertQuery[T, Tslice] {
	q := &ormInsertQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.InsertQuery]{
			BaseQuery: insertTableBaseQuery(t.NameAs(), t.nonGeneratedCols, t.Columns, queryMods),
			Hooks:     &t.InsertQueryHooks,
		},
		Scanner: t.scanner,
	}

	return q.Apply(queryMods...)
}

// Starts an Update query for this table
func (t *Table[T, Tslice, Tset, C]) Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) *ormUpdateQuery[T, Tslice] {
	q := &ormUpdateQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.UpdateQuery]{
			BaseQuery: updateTableBaseQuery(t.NameAs(), t.Columns, queryMods),
			Hooks:     &t.UpdateQueryHooks,
		},
		Scanner: t.scanner,
	}

	return q.Apply(queryMods...)
}

// Starts a Delete query for this table
func (t *Table[T, Tslice, Tset, C]) Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) *ormDeleteQuery[T, Tslice] {
	q := &ormDeleteQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.DeleteQuery]{
			BaseQuery: deleteTableBaseQuery(t.NameAs(), t.Columns, queryMods),
			Hooks:     &t.DeleteQueryHooks,
		},
		Scanner: t.scanner,
	}

	return q.Apply(queryMods...)
}

// Starts a Merge query for this table
// The caller must provide USING and WHEN clauses via queryMods
// RETURNING clause is automatically added if version >= 17 is set in context.
// Use psql.SetVersion(ctx, 17) to enable automatic RETURNING for MERGE.
// For older versions, use mm.Returning() explicitly if needed.
func (t *Table[T, Tslice, Tset, C]) Merge(queryMods ...bob.Mod[*dialect.MergeQuery]) *ormMergeQuery[T, Tslice] {
	q := &ormMergeQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.MergeQuery]{
			BaseQuery: Merge(mm.Into(t.NameAs())),
			Hooks:     &t.MergeQueryHooks,
		},
		Scanner: t.scanner,
	}

	q.Expression.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.MergeQuery) (context.Context, error) {
			// RETURNING in MERGE requires version 17+
			if VersionAtLeast(ctx, 17) && !q.HasReturning() {
				q.AppendReturning(t.Columns)
			}
			return ctx, nil
		},
	)

	q.Apply(queryMods...)

	return q
}

func insertTableBaseQuery(name any, nonGeneratedCols []string, returning bob.Expression, queryMods []bob.Mod[*dialect.InsertQuery]) bob.BaseQuery[*dialect.InsertQuery] {
	base := Insert(im.Into(name, nonGeneratedCols...)).derivedInsertQuery.mutableBase()
	if !hasInsertReturning(queryMods) {
		base.Expression.AppendReturning(orm.DefaultReturning(returning))
	}
	return base
}

func updateTableBaseQuery(name any, returning bob.Expression, queryMods []bob.Mod[*dialect.UpdateQuery]) bob.BaseQuery[*dialect.UpdateQuery] {
	base := Update(um.Table(name)).derivedUpdateQuery.mutableBase()
	if !hasUpdateReturning(queryMods) {
		base.Expression.AppendReturning(orm.DefaultReturning(returning))
	}
	return base
}

func deleteTableBaseQuery(name any, returning bob.Expression, queryMods []bob.Mod[*dialect.DeleteQuery]) bob.BaseQuery[*dialect.DeleteQuery] {
	base := Delete(dm.From(name)).derivedDeleteQuery.mutableBase()
	if !hasDeleteReturning(queryMods) {
		base.Expression.AppendReturning(orm.DefaultReturning(returning))
	}
	return base
}

func hasInsertReturning(mods []bob.Mod[*dialect.InsertQuery]) bool {
	for _, mod := range mods {
		if _, ok := mod.(bobmods.Returning[*dialect.InsertQuery]); ok {
			return true
		}
	}
	return false
}

func hasUpdateReturning(mods []bob.Mod[*dialect.UpdateQuery]) bool {
	for _, mod := range mods {
		if _, ok := mod.(bobmods.Returning[*dialect.UpdateQuery]); ok {
			return true
		}
	}
	return false
}

func hasDeleteReturning(mods []bob.Mod[*dialect.DeleteQuery]) bool {
	for _, mod := range mods {
		if _, ok := mod.(bobmods.Returning[*dialect.DeleteQuery]); ok {
			return true
		}
	}
	return false
}
