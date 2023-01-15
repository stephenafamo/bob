package sqlite

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/dm"
	"github.com/stephenafamo/bob/dialect/sqlite/im"
	"github.com/stephenafamo/bob/dialect/sqlite/um"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

var ErrNothingToUpdate = errors.New("nothing to update")

func NewTable[T any, Topt any](schema, tableName string) *Table[T, []T, Topt] {
	return NewTablex[T, []T, Topt](schema, tableName)
}

func NewTablex[T any, Tslice ~[]T, Topt any](schema, tableName string) *Table[T, Tslice, Topt] {
	var zeroOpt Topt

	optMapping := internal.GetMappings(reflect.TypeOf(zeroOpt))
	view, mappings := newView[T, Tslice](schema, tableName)
	return &Table[T, Tslice, Topt]{
		View:       view,
		pkCols:     internal.FilterNonZero(mappings.PKs),
		optMapping: optMapping,
	}
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Topt any] struct {
	*View[T, Tslice]
	pkCols     []string
	optMapping internal.Mapping

	BeforeInsertHooks orm.Hooks[Topt]
	AfterInsertHooks  orm.Hooks[T]

	BeforeUpsertHooks orm.Hooks[Topt]
	AfterUpsertHooks  orm.Hooks[T]

	BeforeUpdateHooks orm.Hooks[T]
	AfterUpdateHooks  orm.Hooks[T]

	BeforeDeleteHooks orm.Hooks[T]
	AfterDeleteHooks  orm.Hooks[T]
}

// Insert inserts a row into the table with only the set columns in Topt
func (t *Table[T, Tslice, Topt]) Insert(ctx context.Context, exec bob.Executor, row Topt) (T, error) {
	var err error
	var zero T

	ctx, err = t.BeforeInsertHooks.Do(ctx, exec, row)
	if err != nil {
		return zero, nil
	}

	columns, values, err := internal.GetColumnValues(t.optMapping, nil, row)
	if err != nil {
		return zero, fmt.Errorf("get insert values: %w", err)
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Rows(values...),
		im.Returning(t.Columns()),
	)

	val, err := bob.One(ctx, exec, q, scan.StructMapper[T]())
	if err != nil {
		return val, err
	}

	_, err = t.AfterInsertHooks.Do(ctx, exec, val)
	if err != nil {
		return val, err
	}

	return val, nil
}

// Insert inserts a row into the table with only the set columns in Topt
func (t *Table[T, Tslice, Topt]) InsertMany(ctx context.Context, exec bob.Executor, rows ...Topt) (Tslice, error) {
	var err error

	for _, row := range rows {
		ctx, err = t.BeforeInsertHooks.Do(ctx, exec, row)
		if err != nil {
			return nil, nil
		}
	}

	columns, values, err := internal.GetColumnValues(t.optMapping, nil, rows...)
	if err != nil {
		return nil, fmt.Errorf("get insert values: %w", err)
	}

	// If there are no columns, force at least one column with "DEFAULT" for each row
	if len(columns) == 0 {
		columns = []string{internal.FirstNonEmpty(t.optMapping.All)}
		values = make([][]any, len(rows))
		for i := range rows {
			values[i] = []any{"DEFAULT"}
		}
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Rows(values...),
		im.Returning(t.Columns()),
	)

	vals, err := bob.All(ctx, exec, q, scan.StructMapper[T]())
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
func (t *Table[T, Tslice, Topt]) Update(ctx context.Context, exec bob.Executor, row T, cols ...string) (int64, error) {
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
		q.Apply(um.Set(col, values[0][i]))
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
func (t *Table[T, Tslice, Topt]) UpdateMany(ctx context.Context, exec bob.Executor, vals Topt, rows ...T) (int64, error) {
	columns, values, err := internal.GetColumnValues(t.optMapping, nil, vals)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}
	if len(columns) == 0 {
		return 0, ErrNothingToUpdate
	}

	for _, row := range rows {
		_, err = t.BeforeUpdateHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, err
		}
	}

	q := Update(um.Table(t.NameAs(ctx)))

	for i, col := range columns {
		q.Apply(um.Set(col, values[0][i]))
	}

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, rows...)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	pkPairs := make([]any, len(pkVals))
	for i, pair := range pkVals {
		pkPairs[i] = Group(pair...)
	}

	pkGroup := make([]any, len(pks))
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

// Uses the optional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Topt
// if no column is set in Topt (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Topt]) Upsert(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols []string, row Topt) (T, error) {
	var err error
	var zero T

	ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, row)
	if err != nil {
		return zero, nil
	}

	columns, values, err := internal.GetColumnValues(t.optMapping, updateCols, row)
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
		excludeSetCols := columns
		if len(excludeSetCols) == 0 {
			excludeSetCols = t.optMapping.NonPKs
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

	val, err := bob.One(ctx, exec, q, scan.StructMapper[T]())
	if err != nil {
		return val, err
	}

	_, err = t.AfterUpsertHooks.Do(ctx, exec, val)
	if err != nil {
		return val, err
	}

	return val, nil
}

// Uses the optional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Topt
// if no column is set in Topt (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Topt]) UpsertMany(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols []string, rows ...Topt) (Tslice, error) {
	var err error

	for _, row := range rows {
		ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, row)
		if err != nil {
			return nil, nil
		}
	}

	columns, values, err := internal.GetColumnValues(t.optMapping, updateCols, rows...)
	if err != nil {
		return nil, fmt.Errorf("get upsert values: %w", err)
	}

	// If there are no columns, force at least one column with "DEFAULT" for each row
	if len(columns) == 0 {
		columns = []string{internal.FirstNonEmpty(t.optMapping.All)}
		values = make([][]any, len(rows))
		for i := range rows {
			values[i] = []any{"DEFAULT"}
		}
	}

	if len(conflictCols) == 0 {
		conflictCols = t.pkCols
	}

	var conflictQM bob.Mod[*dialect.InsertQuery]
	if !updateOnConflict {
		conflictQM = im.OnConflict(internal.ToAnySlice(conflictCols)...).DoNothing()
	} else {
		excludeSetCols := columns
		if len(excludeSetCols) == 0 {
			excludeSetCols = t.optMapping.NonPKs
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

	vals, err := bob.All(ctx, exec, q, scan.StructMapper[T]())
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
func (t *Table[T, Tslice, Topt]) Delete(ctx context.Context, exec bob.Executor, row T) (int64, error) {
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
func (t *Table[T, Tslice, Topt]) DeleteMany(ctx context.Context, exec bob.Executor, rows ...T) (int64, error) {
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

	pkPairs := make([]any, len(pkVals))
	for i, pair := range pkVals {
		pkPairs[i] = Group(pair...)
	}

	pkGroup := make([]any, len(pks))
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
func (t *Table[T, Tslice, Topt]) Query(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.SelectQuery]) *TableQuery[T, Tslice, Topt] {
	vq := t.View.Query(ctx, exec, queryMods...)
	return &TableQuery[T, Tslice, Topt]{
		ViewQuery:  *vq,
		nameExpr:   t.NameAs,
		pkCols:     t.pkCols,
		optMapping: t.optMapping,
	}
}

type TableQuery[T any, Ts ~[]T, Topt any] struct {
	ViewQuery[T, Ts]
	nameExpr   func(context.Context) bob.Expression // to prevent calling it prematurely
	pkCols     []string
	optMapping internal.Mapping
}

// UpdateAll updates all rows matched by the current query
// NOTE: Hooks cannot be run since the values are never retrieved
func (t *TableQuery[T, Tslice, Topt]) UpdateAll(vals Topt) (int64, error) {
	columns, values, err := internal.GetColumnValues(t.optMapping, nil, vals)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}
	if len(columns) == 0 {
		return 0, ErrNothingToUpdate
	}

	q := Update(um.Table(t.nameExpr(t.ctx)))

	for i, col := range columns {
		q.Apply(um.Set(col, values[0][i]))
	}

	pkGroup := make([]any, len(t.pkCols))
	for i, pk := range t.pkCols {
		pkGroup[i] = Quote(pk)
	}

	// Select ONLY the primary keys
	t.q.Expression.SelectList.Columns = pkGroup
	// WHERE (col1, col2) IN (SELECT ...)
	q.Apply(um.Where(
		Group(pkGroup...).In(t.q.Expression),
	))

	result, err := q.Exec(t.ctx, t.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteAll deletes all rows matched by the current query
// NOTE: Hooks cannot be run since the values are never retrieved
func (t *TableQuery[T, Tslice, Topt]) DeleteAll() (int64, error) {
	q := Delete(dm.From(t.nameExpr(t.ctx)))

	pkGroup := make([]any, len(t.pkCols))
	for i, pk := range t.pkCols {
		pkGroup[i] = Quote(pk)
	}

	// Select ONLY the primary keys
	t.q.Expression.SelectList.Columns = pkGroup
	// WHERE (col1, col2) IN (SELECT ...)
	q.Apply(dm.Where(
		Group(pkGroup...).In(t.q.Expression),
	))

	result, err := q.Exec(t.ctx, t.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
