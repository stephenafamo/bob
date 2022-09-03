package model

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	inqm "github.com/stephenafamo/bob/dialect/psql/insert/qm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

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
// hooks ???
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
// if columns is nil, every column is updated
// NOTE: values from the DB are not refreshed into the model
func (t *Table[T, Tslice, Topt]) Update(ctx context.Context, exec bob.Executor, col *orm.Columns, row T) (int64, error) {
	panic("not implemented")
}

// Updates the given models
// if columns is nil, every column is updated
// NOTE: values from the DB are not refreshed into the models
func (t *Table[T, Tslice, Topt]) UpdateMany(ctx context.Context, exec bob.Executor, vals Topt, rows ...T) (int64, error) {
	panic("not implemented")
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
		inqm.Values(values[0]...),
		inqm.Returning("*"),
		conflictQM,
	)

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
	panic("not implemented")
}

// Deletes the given models
// if columns is nil, every column is deleted
func (t *Table[T, Tslice, Topt]) DeleteMany(ctx context.Context, exec bob.Executor, rows ...T) (int64, error) {
	panic("not implemented")
}

// Adds table name et al
func (t *Table[T, Tslice, Topt]) Query(queryMods ...bob.Mod[*psql.SelectQuery]) *TableQuery[T, Tslice, Topt] {
	vq := t.View.Query(queryMods...)
	return &TableQuery[T, Tslice, Topt]{*vq}
}

type TableQuery[T any, Ts ~[]T, Topt any] struct {
	ViewQuery[T, Ts]
}

func (f *TableQuery[T, Tslice, Topt]) UpdateAll(Topt) (int64, error) {
	panic("not implemented")
}

func (f *TableQuery[T, Tslice, Topt]) DeleteAll() (int64, error) {
	panic("not implemented")
}

func toAnySlice[T any, Ts ~[]T](s Ts) []any {
	ret := make([]any, len(s))
	for i, val := range s {
		ret[i] = val
	}

	return ret
}
