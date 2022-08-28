package model

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	inqm "github.com/stephenafamo/bob/dialect/psql/insert/qm"
	"github.com/stephenafamo/bob/dialect/psql/select/qm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewView[T any, Tslice ~[]T](name0 string, nameX ...string) View[T, Tslice] {
	var zero T

	names := append([]string{name0}, nameX...)
	mappings := internal.GetMappings(reflect.TypeOf(zero))
	allCols := mappings.Columns(names...)

	return View[T, Tslice]{
		name:    names,
		prefix:  names[len(names)-1] + ".",
		mapping: mappings,
		allCols: allCols,
		pkCols:  allCols.Only(mappings.PKs...),
	}
}

type View[T any, Tslice ~[]T] struct {
	prefix string
	name   []string

	mapping internal.Mapping
	allCols orm.Columns
	pkCols  orm.Columns

	AfterSelectHooks orm.Hooks[T]
}

func NewTable[T any, Tslice ~[]T, Topt any](name0 string, nameX ...string) Table[T, Tslice, Topt] {
	var zeroOpt Topt
	optMapping := internal.GetMappings(reflect.TypeOf(zeroOpt))
	view := NewView[T, Tslice](name0, nameX...)
	return Table[T, Tslice, Topt]{
		View:       &view,
		optMapping: optMapping,
	}
}

// The table contains extract information from the struct and contains
// hooks ???
// caches ???
type Table[T any, Tslice ~[]T, Topt any] struct {
	*View[T, Tslice]
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

func (t *View[T, Tslice]) Name() psql.Expression {
	return psql.Quote(t.name...)
}

// Returns a column list
func (t *View[T, Tslice]) Columns() orm.Columns {
	return t.allCols
}

// Returns a column list
func (t *View[T, Tslice]) PKColumns() orm.Columns {
	return t.pkCols
}

// Adds table name et al
func (t *View[T, Tslice]) Query(queryMods ...bob.Mod[*psql.SelectQuery]) *ViewQuery[T, Tslice] {
	q := psql.Select(qm.From(t.Name()))
	q.Apply(queryMods...)

	// Append the table columns
	if len(q.Expression.Select.Columns) == 0 {
		q.Expression.AppendSelect(t.Columns())
	}

	return &ViewQuery[T, Tslice]{
		BaseQuery:        q,
		afterSelectHooks: &t.AfterSelectHooks,
	}
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
		inqm.Values(values[0]...),
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
func (t *Table[T, Tslice, Topt]) Upsert(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols *orm.Columns, row Topt) (T, error) {
	panic("not implemented")
}

// Uses the optional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Topt
func (t *Table[T, Tslice, Topt]) UpsertMany(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols *orm.Columns, rows ...Topt) (Tslice, error) {
	panic("not implemented")
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
