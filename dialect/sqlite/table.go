package sqlite

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/dm"
	"github.com/stephenafamo/bob/dialect/sqlite/im"
	"github.com/stephenafamo/bob/dialect/sqlite/um"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
)

func NewTable[T any, Tset any](schema, tableName string) *Table[T, []T, Tset] {
	return NewTablex[T, []T, Tset](schema, tableName)
}

func NewTablex[T any, Tslice ~[]T, Tset any](schema, tableName string) *Table[T, Tslice, Tset] {
	var zeroSet Tset

	setMapping := internal.GetMappings(reflect.TypeOf(zeroSet))
	view, mappings := newView[T, Tslice](schema, tableName)
	return &Table[T, Tslice, Tset]{
		View:       view,
		pkCols:     internal.FilterNonZero(mappings.PKs),
		setMapping: setMapping,
	}
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset any] struct {
	*View[T, Tslice]
	pkCols     []string
	setMapping internal.Mapping

	BeforeInsertHooks orm.Hooks[Tset]
	AfterInsertHooks  orm.Hooks[T]

	BeforeUpsertHooks orm.Hooks[Tset]
	AfterUpsertHooks  orm.Hooks[T]

	BeforeUpdateHooks orm.Hooks[T]
	AfterUpdateHooks  orm.Hooks[T]

	BeforeDeleteHooks orm.Hooks[T]
	AfterDeleteHooks  orm.Hooks[T]
}

// Insert inserts a row into the table with only the set columns in Tset
func (t *Table[T, Tslice, Tset]) Insert(ctx context.Context, exec bob.Executor, row Tset) (T, error) {
	var err error
	var zero T

	ctx, err = t.BeforeInsertHooks.Do(ctx, exec, row)
	if err != nil {
		return zero, err
	}

	columns, values, err := internal.GetColumnValues(t.setMapping, nil, row)
	if err != nil {
		return zero, fmt.Errorf("get insert values: %w", err)
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Rows(values...),
		im.Returning(t.Columns()),
	)

	val, err := bob.One(ctx, exec, q, t.scanner)
	if err != nil {
		return val, err
	}

	_, err = t.AfterInsertHooks.Do(ctx, exec, val)
	if err != nil {
		return val, err
	}

	return val, nil
}

// Insert inserts a row into the table with only the set columns in Tset
func (t *Table[T, Tslice, Tset]) InsertMany(ctx context.Context, exec bob.Executor, rows ...Tset) (Tslice, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	var err error

	for _, row := range rows {
		ctx, err = t.BeforeInsertHooks.Do(ctx, exec, row)
		if err != nil {
			return nil, err
		}
	}

	columns, values, err := internal.GetColumnValues(t.setMapping, nil, rows...)
	if err != nil {
		return nil, fmt.Errorf("get insert values: %w", err)
	}

	// If there are no columns, force at least one column with "DEFAULT" for each row
	if len(columns) == 0 {
		columns = []string{internal.FirstNonEmpty(t.setMapping.All)}
		values = make([][]bob.Expression, len(rows))
		for i := range rows {
			values[i] = []bob.Expression{Raw("DEFAULT")}
		}
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Rows(values...),
		im.Returning(t.Columns()),
	)

	vals, err := bob.All(ctx, exec, q, t.scanner)
	if err != nil {
		return vals, err
	}

	for _, val := range vals {
		_, err = t.AfterInsertHooks.Do(ctx, exec, val)
		if err != nil {
			return vals, err
		}
	}

	return vals, nil
}

// Updates the given model
// if columns is nil, every non-primary-key column is updated
// NOTE: values from the DB are not refreshed into the model
func (t *Table[T, Tslice, Tset]) Update(ctx context.Context, exec bob.Executor, row T, cols ...string) (int64, error) {
	_, err := t.BeforeUpdateHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	q := Update(um.Table(t.NameAs(ctx)))

	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, row)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	if len(cols) == 0 {
		cols = t.mapping.NonPKs
	}
	columns, values, err := internal.GetColumnValues(t.mapping, cols, row)
	if err != nil {
		return 0, fmt.Errorf("get update values: %w", err)
	}

	for i, pk := range pks {
		q.Apply(um.Where(Quote(pk).EQ(pkVals[0][i])))
	}

	for i, col := range columns {
		q.Apply(um.Set(col).To(values[0][i]))
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterUpdateHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Updates the given models
// if columns is nil, every column is updated
// NOTE: values from the DB are not refreshed into the models
func (t *Table[T, Tslice, Tset]) UpdateMany(ctx context.Context, exec bob.Executor, vals Tset, rows ...T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	columns, values, err := internal.GetColumnValues(t.setMapping, nil, vals)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}
	if len(columns) == 0 {
		return 0, orm.ErrNothingToUpdate
	}

	for _, row := range rows {
		_, err = t.BeforeUpdateHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, err
		}
	}

	q := Update(um.Table(t.NameAs(ctx)))

	for i, col := range columns {
		q.Apply(um.Set(col).To(values[0][i]))
	}

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, rows...)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	pkPairs := make([]bob.Expression, len(pkVals))
	for i, pair := range pkVals {
		pkPairs[i] = Group(pair...)
	}

	pkGroup := make([]bob.Expression, len(pks))
	for i, pk := range pks {
		pkGroup[i] = Quote(pk)
	}

	q.Apply(um.Where(
		Group(pkGroup...).In(pkPairs...),
	))

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	for _, row := range rows {
		_, err = t.AfterUpdateHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, err
		}
	}

	return result.RowsAffected()
}

// Uses the setional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Tset
// if no column is set in Tset (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Tset]) Upsert(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols []string, row Tset) (T, error) {
	var err error
	var zero T

	ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, row)
	if err != nil {
		return zero, err
	}

	columns, values, err := internal.GetColumnValues(t.setMapping, nil, row)
	if err != nil {
		return zero, fmt.Errorf("get upsert values: %w", err)
	}

	if len(conflictCols) == 0 {
		conflictCols = t.pkCols
	}

	var conflictQM bob.Mod[*dialect.InsertQuery]
	if !updateOnConflict {
		conflictQM = im.OnConflict(internal.ToAnySlice(conflictCols)...).DoNothing()
	} else {
		excludeSetCols := updateCols
		// If no update columns, use the columns set
		if len(excludeSetCols) == 0 {
			excludeSetCols = columns
		}
		// if still empty, use non-PKs
		if len(excludeSetCols) == 0 {
			excludeSetCols = t.setMapping.NonPKs
		}
		conflictQM = im.OnConflict(internal.ToAnySlice(conflictCols)...).
			DoUpdate().
			SetExcluded(excludeSetCols...)
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Rows(values...),
		im.Returning(t.Columns()),
		conflictQM,
	)

	val, err := bob.One(ctx, exec, q, t.scanner)
	if err != nil {
		return val, err
	}

	_, err = t.AfterUpsertHooks.Do(ctx, exec, val)
	if err != nil {
		return val, err
	}

	return val, nil
}

// Uses the setional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Tset
// if no column is set in Tset (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Tset]) UpsertMany(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols []string, rows ...Tset) (Tslice, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	var err error

	for _, row := range rows {
		ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, row)
		if err != nil {
			return nil, err
		}
	}

	columns, values, err := internal.GetColumnValues(t.setMapping, nil, rows...)
	if err != nil {
		return nil, fmt.Errorf("get upsert values: %w", err)
	}

	// If there are no columns, force at least one column with "DEFAULT" for each row
	if len(columns) == 0 {
		columns = []string{internal.FirstNonEmpty(t.setMapping.All)}
		values = make([][]bob.Expression, len(rows))
		for i := range rows {
			values[i] = []bob.Expression{Raw("DEFAULT")}
		}
	}

	if len(conflictCols) == 0 {
		conflictCols = t.pkCols
	}

	var conflictQM bob.Mod[*dialect.InsertQuery]
	if !updateOnConflict {
		conflictQM = im.OnConflict(internal.ToAnySlice(conflictCols)...).DoNothing()
	} else {
		excludeSetCols := updateCols
		// If no update columns, use the columns set
		if len(excludeSetCols) == 0 {
			excludeSetCols = columns
		}
		// if still empty, use non-PKs
		if len(excludeSetCols) == 0 {
			excludeSetCols = t.setMapping.NonPKs
		}
		conflictQM = im.OnConflict(internal.ToAnySlice(conflictCols)...).
			DoUpdate().
			SetExcluded(excludeSetCols...)
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Returning(t.Columns()),
		conflictQM,
	)

	for _, val := range values {
		q.Apply(im.Values(val...))
	}

	vals, err := bob.All(ctx, exec, q, t.scanner)
	if err != nil {
		return vals, err
	}

	for _, val := range vals {
		_, err = t.AfterUpsertHooks.Do(ctx, exec, val)
		if err != nil {
			return nil, err
		}
	}

	return vals, nil
}

// Deletes the given model
// if columns is nil, every column is deleted
func (t *Table[T, Tslice, Tset]) Delete(ctx context.Context, exec bob.Executor, row T) (int64, error) {
	_, err := t.BeforeDeleteHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	q := Delete(dm.From(t.NameAs(ctx)))

	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, row)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	for i, pk := range pks {
		q.Apply(dm.Where(Quote(pk).EQ(pkVals[0][i])))
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterDeleteHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Deletes the given models
// if columns is nil, every column is deleted
func (t *Table[T, Tslice, Tset]) DeleteMany(ctx context.Context, exec bob.Executor, rows ...T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	for _, row := range rows {
		_, err := t.BeforeDeleteHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, err
		}
	}

	q := Delete(dm.From(t.NameAs(ctx)))

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, rows...)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	pkPairs := make([]bob.Expression, len(pkVals))
	for i, pair := range pkVals {
		pkPairs[i] = Group(pair...)
	}

	pkGroup := make([]bob.Expression, len(pks))
	for i, pk := range pks {
		pkGroup[i] = Quote(pk)
	}

	q.Apply(dm.Where(
		Group(pkGroup...).In(pkPairs...),
	))

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	for _, row := range rows {
		_, err = t.AfterDeleteHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, err
		}
	}

	return result.RowsAffected()
}

// Adds table name et al
func (t *Table[T, Tslice, Tset]) Query(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.SelectQuery]) *TableQuery[T, Tslice, Tset] {
	vq := t.View.Query(ctx, exec, queryMods...)
	return &TableQuery[T, Tslice, Tset]{
		ViewQuery:  *vq,
		nameExpr:   t.NameAs,
		pkCols:     t.pkCols,
		setMapping: t.setMapping,
	}
}

type TableQuery[T any, Ts ~[]T, Tset any] struct {
	ViewQuery[T, Ts]
	nameExpr   func(context.Context) bob.Expression // to prevent calling it prematurely
	pkCols     []string
	setMapping internal.Mapping
}

// UpdateAll updates all rows matched by the current query
// NOTE: Hooks cannot be run since the values are never retrieved
func (t *TableQuery[T, Tslice, Tset]) UpdateAll(vals Tset) (int64, error) {
	columns, values, err := internal.GetColumnValues(t.setMapping, nil, vals)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}
	if len(columns) == 0 {
		return 0, orm.ErrNothingToUpdate
	}

	q := Update(um.Table(t.nameExpr(t.ctx)))

	for i, col := range columns {
		q.Apply(um.Set(col).To(values[0][i]))
	}

	pkCols := make([]any, len(t.pkCols))
	pkGroup := make([]bob.Expression, len(t.pkCols))
	for i, pk := range t.pkCols {
		q := Quote(pk)
		pkGroup[i] = q
		pkCols[i] = q
	}

	// Select ONLY the primary keys
	t.BaseQuery.Expression.SelectList.Columns = pkCols
	// WHERE (col1, col2) IN (SELECT ...)
	q.Apply(um.Where(
		Group(pkGroup...).In(t.BaseQuery.Expression),
	))

	result, err := q.Exec(t.ctx, t.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteAll deletes all rows matched by the current query
// NOTE: Hooks cannot be run since the values are never retrieved
func (t *TableQuery[T, Tslice, Tset]) DeleteAll() (int64, error) {
	q := Delete(dm.From(t.nameExpr(t.ctx)))

	pkCols := make([]any, len(t.pkCols))
	pkGroup := make([]bob.Expression, len(t.pkCols))
	for i, pk := range t.pkCols {
		q := Quote(pk)
		pkGroup[i] = q
		pkCols[i] = q
	}

	// Select ONLY the primary keys
	t.BaseQuery.Expression.SelectList.Columns = pkCols
	// WHERE (col1, col2) IN (SELECT ...)
	q.Apply(dm.Where(
		Group(pkGroup...).In(t.BaseQuery.Expression),
	))

	result, err := q.Exec(t.ctx, t.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
