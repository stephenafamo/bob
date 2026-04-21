package psql

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/mm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	bobmods "github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/orm"
)

type (
	setter[T any]                     = orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]
	ormMergeQuery[T any, Tslice ~[]T] = orm.Query[*dialect.MergeQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
)

type ormInsertQuery[T any, Tslice ~[]T] struct {
	orm.Query[*dialect.InsertQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	defaultReturning bob.Expression
}

type ormUpdateQuery[T any, Tslice ~[]T] struct {
	orm.Query[*dialect.UpdateQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	defaultReturning bob.Expression
}

type ormDeleteQuery[T any, Tslice ~[]T] struct {
	orm.Query[*dialect.DeleteQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]
	defaultReturning bob.Expression
}

func (q ormInsertQuery[T, Tslice]) clone() ormInsertQuery[T, Tslice] {
	return ormInsertQuery[T, Tslice]{
		Query:            q.Query.Clone(),
		defaultReturning: q.defaultReturning,
	}
}

func (q *ormInsertQuery[T, Tslice]) With(queryMods ...bob.Mod[*dialect.InsertQuery]) *ormInsertQuery[T, Tslice] {
	if q == nil {
		return nil
	}

	next := q.clone()
	applyTableQueryMods(next.Expression, next.defaultReturning, func(query *dialect.InsertQuery) *clause.Returning {
		return &query.Returning
	}, queryMods...)
	return &next
}

func (q *ormInsertQuery[T, Tslice]) Apply(queryMods ...bob.Mod[*dialect.InsertQuery]) *ormInsertQuery[T, Tslice] {
	return q.With(queryMods...)
}

func (q ormUpdateQuery[T, Tslice]) clone() ormUpdateQuery[T, Tslice] {
	return ormUpdateQuery[T, Tslice]{
		Query:            q.Query.Clone(),
		defaultReturning: q.defaultReturning,
	}
}

func (q *ormUpdateQuery[T, Tslice]) With(queryMods ...bob.Mod[*dialect.UpdateQuery]) *ormUpdateQuery[T, Tslice] {
	if q == nil {
		return nil
	}

	next := q.clone()
	applyTableQueryMods(next.Expression, next.defaultReturning, func(query *dialect.UpdateQuery) *clause.Returning {
		return &query.Returning
	}, queryMods...)
	return &next
}

func (q *ormUpdateQuery[T, Tslice]) Apply(queryMods ...bob.Mod[*dialect.UpdateQuery]) *ormUpdateQuery[T, Tslice] {
	return q.With(queryMods...)
}

func (q ormDeleteQuery[T, Tslice]) clone() ormDeleteQuery[T, Tslice] {
	return ormDeleteQuery[T, Tslice]{
		Query:            q.Query.Clone(),
		defaultReturning: q.defaultReturning,
	}
}

func (q *ormDeleteQuery[T, Tslice]) With(queryMods ...bob.Mod[*dialect.DeleteQuery]) *ormDeleteQuery[T, Tslice] {
	if q == nil {
		return nil
	}

	next := q.clone()
	applyTableQueryMods(next.Expression, next.defaultReturning, func(query *dialect.DeleteQuery) *clause.Returning {
		return &query.Returning
	}, queryMods...)
	return &next
}

func (q *ormDeleteQuery[T, Tslice]) Apply(queryMods ...bob.Mod[*dialect.DeleteQuery]) *ormDeleteQuery[T, Tslice] {
	return q.With(queryMods...)
}

func applyTableQueryMods[Q interface{ AppendReturning(...any) }](query Q, defaultReturning bob.Expression, getReturning func(Q) *clause.Returning, queryMods ...bob.Mod[Q]) {
	if hasExplicitReturning(queryMods...) && hasOnlyDefaultReturning(getReturning(query).Expressions, defaultReturning) {
		getReturning(query).Expressions = nil
	}

	for _, mod := range queryMods {
		mod.Apply(query)
	}
}

func hasExplicitReturning[Q interface{ AppendReturning(...any) }](queryMods ...bob.Mod[Q]) bool {
	for _, mod := range queryMods {
		if _, ok := mod.(bobmods.Returning[Q]); ok {
			return true
		}
	}

	return false
}

func hasOnlyDefaultReturning(expressions []any, defaultReturning bob.Expression) bool {
	return len(expressions) == 1 && reflect.DeepEqual(expressions[0], defaultReturning)
}

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
		Query: orm.Query[*dialect.InsertQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]{
			ExecQuery: orm.ExecQuery[*dialect.InsertQuery]{
				BaseQuery: insertTableBaseQuery(t.NameAs(), t.nonGeneratedCols, t.Columns),
				Hooks:     &t.InsertQueryHooks,
			},
			Scanner: t.scanner,
		},
		defaultReturning: t.Columns,
	}

	return q.Apply(queryMods...)
}

// Starts an Update query for this table
func (t *Table[T, Tslice, Tset, C]) Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) *ormUpdateQuery[T, Tslice] {
	q := &ormUpdateQuery[T, Tslice]{
		Query: orm.Query[*dialect.UpdateQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]{
			ExecQuery: orm.ExecQuery[*dialect.UpdateQuery]{
				BaseQuery: updateTableBaseQuery(t.NameAs(), t.Columns),
				Hooks:     &t.UpdateQueryHooks,
			},
			Scanner: t.scanner,
		},
		defaultReturning: t.Columns,
	}

	return q.Apply(queryMods...)
}

// Starts a Delete query for this table
func (t *Table[T, Tslice, Tset, C]) Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) *ormDeleteQuery[T, Tslice] {
	q := &ormDeleteQuery[T, Tslice]{
		Query: orm.Query[*dialect.DeleteQuery, T, Tslice, bob.SliceTransformer[T, Tslice]]{
			ExecQuery: orm.ExecQuery[*dialect.DeleteQuery]{
				BaseQuery: deleteTableBaseQuery(t.NameAs(), t.Columns),
				Hooks:     &t.DeleteQueryHooks,
			},
			Scanner: t.scanner,
		},
		defaultReturning: t.Columns,
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

func insertTableBaseQuery(name any, nonGeneratedCols []string, returning bob.Expression) bob.BaseQuery[*dialect.InsertQuery] {
	base := bob.BaseQuery[*dialect.InsertQuery]{
		Expression: &dialect.InsertQuery{
			TableRef: clause.TableRef{
				Expression: name,
				Columns:    nonGeneratedCols,
			},
		},
		Dialect:   dialect.Dialect,
		QueryType: bob.QueryTypeInsert,
	}
	base.Expression.AppendReturning(returning)
	return base
}

func updateTableBaseQuery(name any, returning bob.Expression) bob.BaseQuery[*dialect.UpdateQuery] {
	base := bob.BaseQuery[*dialect.UpdateQuery]{
		Expression: &dialect.UpdateQuery{
			Table: clause.TableRef{
				Expression: name,
			},
		},
		Dialect:   dialect.Dialect,
		QueryType: bob.QueryTypeUpdate,
	}
	base.Expression.AppendReturning(returning)
	return base
}

func deleteTableBaseQuery(name any, returning bob.Expression) bob.BaseQuery[*dialect.DeleteQuery] {
	base := bob.BaseQuery[*dialect.DeleteQuery]{
		Expression: &dialect.DeleteQuery{
			Table: clause.TableRef{
				Expression: name,
			},
		},
		Dialect:   dialect.Dialect,
		QueryType: bob.QueryTypeDelete,
	}
	base.Expression.AppendReturning(returning)
	return base
}
