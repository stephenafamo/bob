package psql

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/mm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/internal/ormruntime"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type (
	modelRows[Q any, T any, S ~[]T] = orm.Query[Q, T, S, bob.SliceTransformer[T, S]]
	rowOperations[T any, S ~[]T]    = ormruntime.Rows[T, S, *ViewQuery[T, S]]
	setter[T any]                   = orm.Setter[T, dialect.InsertEditor, dialect.UpdateEditor]
)

func NewTable[T any, Tset setter[T], C bob.Expression](
	schema, tableName string, columns C, rows orm.RowMapping[T, []T],
) *Table[T, []T, Tset, C] {
	return NewTablex[T, []T, Tset](schema, tableName, columns, nil, rows)
}

// NewTablex creates a Table with a row mapping and a custom scanner.
// If scanner is nil, it falls back to [scan.StructMapper].
func NewTablex[T any, Tslice ~[]T, Tset setter[T], C bob.Expression](
	schema, table string,
	columns C,
	scanner scan.Mapper[T],
	rows orm.RowMapping[T, Tslice],
) *Table[T, Tslice, Tset, C] {
	view := NewViewx[T, Tslice](schema, table, columns, scanner)
	return &Table[T, Tslice, Tset, C]{
		View:             view,
		rowOperations:    ormruntime.NewRows(rows, view.lookup),
		nonGeneratedCols: mappings.WriteColumns[T](scanner != nil, view.ColumnsExpr().Names()).NonGenerated,
	}
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset setter[T], C bob.Expression] struct {
	*View[T, Tslice, C]
	rowOperations[T, Tslice]
	nonGeneratedCols []string

	BeforeInsertHooks bob.Hooks[Tset, bob.SkipModelHooksKey]
	AfterInsertHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeUpdateHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterUpdateHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeDeleteHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterDeleteHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeMergeHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterMergeHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	InsertQueryHooks bob.QueryHooks[InsertEditor]
	UpdateQueryHooks bob.QueryHooks[UpdateEditor]
	DeleteQueryHooks bob.QueryHooks[DeleteEditor]
	MergeQueryHooks  bob.QueryHooks[MergeEditor]
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset, C]) Insert(queryMods ...bob.Mod[dialect.InsertEditor]) *modelRows[dialect.InsertEditor, T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewInsert(t.InsertQueryHooks.GetHooks(), bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) {
		im.Into(t.NameAsExpr(), t.nonGeneratedCols...).Edit(q)

		q.AppendContextualMod(dialect.DefaultReturning[dialect.InsertEditor]{Columns: t.Columns})

		bob.Mods[dialect.InsertEditor](queryMods).Edit(q)
	})), t.scanner)
}

// Starts an Update query for this table
func (t *Table[T, Tslice, Tset, C]) Update(queryMods ...bob.Mod[dialect.UpdateEditor]) *modelRows[dialect.UpdateEditor, T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewUpdate(t.UpdateQueryHooks.GetHooks(), bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
		um.Table(t.NameAsExpr()).Edit(q)

		q.AppendContextualMod(dialect.DefaultReturning[dialect.UpdateEditor]{Columns: t.Columns})

		bob.Mods[dialect.UpdateEditor](queryMods).Edit(q)
	})), t.scanner)
}

// Starts a Delete query for this table
func (t *Table[T, Tslice, Tset, C]) Delete(queryMods ...bob.Mod[dialect.DeleteEditor]) *modelRows[dialect.DeleteEditor, T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewDelete(t.DeleteQueryHooks.GetHooks(), bob.ModFunc[dialect.DeleteEditor](func(q dialect.DeleteEditor) {
		dm.From(t.NameAsExpr()).Edit(q)

		q.AppendContextualMod(dialect.DefaultReturning[dialect.DeleteEditor]{Columns: t.Columns})

		bob.Mods[dialect.DeleteEditor](queryMods).Edit(q)
	})), t.scanner)
}

// Starts a Merge query for this table
// The caller must provide USING and WHEN clauses via queryMods
// RETURNING clause is automatically added if version >= 17 is set in context.
// Use psql.SetVersion(ctx, 17) to enable automatic RETURNING for MERGE.
// For older versions, use mm.Returning() explicitly if needed.
func (t *Table[T, Tslice, Tset, C]) Merge(queryMods ...bob.Mod[dialect.MergeEditor]) *modelRows[dialect.MergeEditor, T, Tslice] {
	return ormruntime.ModelQuery[Tslice](dialect.NewMerge(t.MergeQueryHooks.GetHooks(), bob.ModFunc[dialect.MergeEditor](func(q dialect.MergeEditor) {
		mm.Into(t.NameAsExpr()).Edit(q)

		q.AppendContextualMod(dialect.DefaultReturning[dialect.MergeEditor]{
			Columns: t.Columns, Enabled: func(ctx context.Context) bool { return VersionAtLeast(ctx, 17) },
		})

		bob.Mods[dialect.MergeEditor](queryMods).Edit(q)
	})), t.scanner)
}
