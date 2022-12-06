package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	delqm "github.com/stephenafamo/bob/dialect/psql/delete/qm"
	inqm "github.com/stephenafamo/bob/dialect/psql/insert/qm"
	upqm "github.com/stephenafamo/bob/dialect/psql/update/qm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

var ErrNothingToUpdate = errors.New("nothing to update")

func NewTable[T any, Tslice ~[]T, Topt any](name0 string, nameX ...string) Table[T, Tslice, Topt] {
	var zeroOpt Topt

	optMapping := internal.GetMappings(reflect.TypeOf(zeroOpt))
	view := NewView[T, Tslice](name0, nameX...)
	return Table[T, Tslice, Topt]{
		View:       &view,
		optMapping: optMapping,
		optPkCols:  optMapping.Columns(view.name...).Only(optMapping.PKs...),
	}
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Topt any] struct {
	*View[T, Tslice]
	optPkCols  orm.Columns
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

// Returns a column list
func (t *Table[T, Tslice, Topt]) PKColumns() orm.Columns {
	return t.pkCols
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

	q := psql.Insert(
		inqm.Into(t.Name(), columns...),
		inqm.Rows(values...),
		inqm.Returning("*"),
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
		columns = []string{firstNonEmpty(t.optMapping.All)}
		values = make([][]any, len(rows))
		for i := range rows {
			values[i] = []any{"DEFAULT"}
		}
	}

	q := psql.Insert(
		inqm.Into(t.Name(), columns...),
		inqm.Rows(values...),
		inqm.Returning("*"),
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
func (t *Table[T, Tslice, Topt]) Update(ctx context.Context, exec bob.Executor, cols []string, row T) (int64, error) {
	_, err := t.BeforeUpdateHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	q := psql.Update(upqm.Table(t.Name()))

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
		q.Apply(upqm.Where(psql.Quote(pk).EQ(pkVals[0][i])))
	}

	for i, col := range columns {
		q.Apply(upqm.Set(col, values[0][i]))
	}

	rowsAff, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterUpdateHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	return rowsAff, nil
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

	q := psql.Update(upqm.Table(t.Name()))

	for i, col := range columns {
		q.Apply(upqm.Set(col, values[0][i]))
	}

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, rows...)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	pkPairs := make([]any, len(pkVals))
	for i, pair := range pkVals {
		pkPairs[i] = psql.Group(pair...)
	}

	pkGroup := make([]any, len(pks))
	for i, pk := range pks {
		pkGroup[i] = psql.Quote(pk)
	}

	q.Apply(upqm.Where(
		psql.Group(pkGroup...).In(pkPairs...),
	))

	rowsAff, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return rowsAff, err
	}

	for _, row := range rows {
		_, err = t.AfterUpdateHooks.Do(ctx, exec, row)
		if err != nil {
			return rowsAff, err
		}
	}

	return rowsAff, nil
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
		conflictCols = t.optPkCols.Names()
	}

	var conflictQM bob.Mod[*psql.InsertQuery]
	if !updateOnConflict {
		conflictQM = inqm.OnConflict(toAnySlice(conflictCols)...).DoNothing()
	} else {
		excludeSetCols := columns
		if len(excludeSetCols) == 0 {
			excludeSetCols = t.optMapping.NonPKs
		}
		conflictQM = inqm.OnConflict(toAnySlice(conflictCols)...).
			DoUpdate().
			SetExcluded(excludeSetCols...)
	}

	q := psql.Insert(
		inqm.Into(t.Name(), columns...),
		inqm.Rows(values...),
		inqm.Returning("*"),
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
		columns = []string{firstNonEmpty(t.optMapping.All)}
		values = make([][]any, len(rows))
		for i := range rows {
			values[i] = []any{"DEFAULT"}
		}
	}

	if len(conflictCols) == 0 {
		conflictCols = t.optPkCols.Names()
	}

	var conflictQM bob.Mod[*psql.InsertQuery]
	if !updateOnConflict {
		conflictQM = inqm.OnConflict(toAnySlice(conflictCols)...).DoNothing()
	} else {
		excludeSetCols := columns
		if len(excludeSetCols) == 0 {
			excludeSetCols = t.optMapping.NonPKs
		}
		conflictQM = inqm.OnConflict(toAnySlice(conflictCols)...).
			DoUpdate().
			SetExcluded(excludeSetCols...)
	}

	q := psql.Insert(
		inqm.Into(t.Name(), columns...),
		inqm.Returning("*"),
		conflictQM,
	)

	for _, val := range values {
		q.Apply(inqm.Values(val...))
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

	q := psql.Delete(delqm.From(t.Name()))

	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, row)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	for i, pk := range pks {
		q.Apply(delqm.Where(psql.Quote(pk).EQ(pkVals[0][i])))
	}

	rowsAff, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterDeleteHooks.Do(ctx, exec, row)
	if err != nil {
		return 0, err
	}

	return rowsAff, nil
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

	q := psql.Delete(delqm.From(t.Name()))

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping, t.mapping.PKs, rows...)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	pkPairs := make([]any, len(pkVals))
	for i, pair := range pkVals {
		pkPairs[i] = psql.Group(pair...)
	}

	pkGroup := make([]any, len(pks))
	for i, pk := range pks {
		pkGroup[i] = psql.Quote(pk)
	}

	q.Apply(delqm.Where(
		psql.Group(pkGroup...).In(pkPairs...),
	))

	rowsAff, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return rowsAff, err
	}

	for _, row := range rows {
		_, err = t.AfterDeleteHooks.Do(ctx, exec, row)
		if err != nil {
			return rowsAff, err
		}
	}

	return rowsAff, nil
}

// Adds table name et al
func (t *Table[T, Tslice, Topt]) Query(queryMods ...bob.Mod[*psql.SelectQuery]) *TableQuery[T, Tslice, Topt] {
	vq := t.View.Query(queryMods...)
	return &TableQuery[T, Tslice, Topt]{
		ViewQuery:  *vq,
		name:       t.name,
		pkCols:     t.pkCols,
		optMapping: t.optMapping,
	}
}

type TableQuery[T any, Ts ~[]T, Topt any] struct {
	ViewQuery[T, Ts]
	name       []string
	pkCols     orm.Columns
	optMapping internal.Mapping
}

// UpdateAll updates all rows matched by the current query
// NOTE: Hooks cannot be run since the values are never retrieved
func (t *TableQuery[T, Tslice, Topt]) UpdateAll(ctx context.Context, exec bob.Executor, vals Topt) (int64, error) {
	columns, values, err := internal.GetColumnValues(t.optMapping, nil, vals)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}
	if len(columns) == 0 {
		return 0, ErrNothingToUpdate
	}

	q := psql.Update(upqm.Table(psql.Quote(t.name...)))

	for i, col := range columns {
		q.Apply(upqm.Set(col, values[0][i]))
	}

	pkGroup := make([]any, len(t.pkCols.Names()))
	for i, pk := range t.pkCols.Names() {
		pkGroup[i] = psql.Quote(pk)
	}

	// Select ONLY the primary keys
	t.Expression.SelectList.Columns = pkGroup
	// WHERE (col1, col2) IN (SELECT ...)
	q.Apply(upqm.Where(
		psql.Group(pkGroup...).In(t.Expression),
	))

	rowsAff, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	return rowsAff, nil
}

// DeleteAll deletes all rows matched by the current query
// NOTE: Hooks cannot be run since the values are never retrieved
func (t *TableQuery[T, Tslice, Topt]) DeleteAll(ctx context.Context, exec bob.Executor) (int64, error) {
	q := psql.Delete(delqm.From(psql.Quote(t.name...)))

	pkGroup := make([]any, len(t.pkCols.Names()))
	for i, pk := range t.pkCols.Names() {
		pkGroup[i] = psql.Quote(pk)
	}

	// Select ONLY the primary keys
	t.Expression.SelectList.Columns = pkGroup
	// WHERE (col1, col2) IN (SELECT ...)
	q.Apply(delqm.Where(
		psql.Group(pkGroup...).In(t.Expression),
	))

	rowsAff, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return 0, err
	}

	return rowsAff, nil
}

func toAnySlice[T any, Ts ~[]T](slice Ts) []any {
	ret := make([]any, len(slice))
	for i, val := range slice {
		ret[i] = val
	}

	return ret
}

func firstNonEmpty[T comparable, Ts ~[]T](slice Ts) T {
	var zero T
	for _, val := range slice {
		if val != zero {
			return val
		}
	}

	return zero
}
