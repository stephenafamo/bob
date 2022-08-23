package model

import (
	"context"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewTable[T any, Tslice ~[]T, Topt any](name0 string, nameX ...string) Table[T, Tslice, Topt] {
	view := NewView[T, Tslice](name0, nameX...)
	return Table[T, Tslice, Topt]{view}
}

func NewView[T any, Tslice ~[]T](name0 string, nameX ...string) View[T, Tslice] {
	var zero T

	names := append([]string{name0}, nameX...)
	cols := internal.GetColumns(reflect.TypeOf(zero))
	allCols := cols.Get(names...)

	return View[T, Tslice]{
		name:    names,
		prefix:  names[len(names)-1] + ".",
		cols:    cols,
		allCols: allCols,
		pkCols:  allCols.Only(cols.PKs...),
	}
}

type View[T any, Tslice ~[]T] struct {
	prefix string
	name   []string

	cols    internal.Columns
	allCols orm.Columns
	pkCols  orm.Columns
}

// The table contains extract information from the struct and contains
// hooks ???
// caches ???
type Table[T any, Tslice ~[]T, Topt any] struct {
	View[T, Tslice]
}

func (t View[T, Tslice]) Name() psql.Expression {
	return psql.Quote(t.name...)
}

// Returns a column list
func (t View[T, Tslice]) Columns() orm.Columns {
	return t.allCols
}

// Returns a column list
func (t View[T, Tslice]) PKColumns() orm.Columns {
	return t.pkCols
}

// Adds table name et al
func (t View[T, Tslice]) Query(queryMods ...bob.Mod[*psql.SelectQuery]) *ViewQuery[T, Tslice] {
	f := &ViewQuery[T, Tslice]{BaseQuery: psql.Select(
		psql.SelectQM.From(t.Name()),
	)}
	f.Apply(queryMods...)

	// Append the table columns
	if len(f.BaseQuery.Expression.Select.Columns) == 0 {
		f.BaseQuery.Expression.AppendSelect(t.Columns())
	}

	return f
}

// Insert inserts a row into the table with only the set columns in Topt
func (t Table[T, Tslice, Topt]) Insert(ctx context.Context, exec scan.Queryer, row Topt) (T, error) {
	panic("not implemented")
}

// Insert inserts a row into the table with only the set columns in Topt
func (t Table[T, Tslice, Topt]) InsertMany(ctx context.Context, exec scan.Queryer, rows ...Topt) (Tslice, error) {
	panic("not implemented")
}

// Updates the given model
// if columns is nil, every column is updated
func (t Table[T, Tslice, Topt]) Update(ctx context.Context, exec scan.Queryer, columns *orm.Columns, row T) (T, int64, error) {
	// should return the updated row
	panic("not implemented")
}

// Updates the given models
// if columns is nil, every column is updated
func (t Table[T, Tslice, Topt]) UpdateMany(ctx context.Context, exec scan.Queryer, vals Topt, rows ...T) (Tslice, int64, error) {
	panic("not implemented")
}

// Uses the optional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Topt
func (t Table[T, Tslice, Topt]) Upsert(ctx context.Context, exec scan.Queryer, updateOnConflict bool, conflictCols, updateCols *orm.Columns, row Topt) (T, error) {
	panic("not implemented")
}

// Uses the optional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Topt
func (t Table[T, Tslice, Topt]) UpsertMany(ctx context.Context, exec scan.Queryer, updateOnConflict bool, conflictCols, updateCols *orm.Columns, rows ...Topt) (Tslice, error) {
	panic("not implemented")
}

// Deletes the given model
// if columns is nil, every column is deleted
func (t Table[T, Tslice, Topt]) Delete(ctx context.Context, exec scan.Queryer, row T) (int64, error) {
	panic("not implemented")
}

// Deletes the given models
// if columns is nil, every column is deleted
func (t Table[T, Tslice, Topt]) DeleteMany(ctx context.Context, exec scan.Queryer, rows ...T) (int64, error) {
	panic("not implemented")
}

// Adds table name et al
func (t Table[T, Tslice, Topt]) Query(queryMods ...bob.Mod[*psql.SelectQuery]) *TableQuery[T, Tslice, Topt] {
	vq := t.View.Query(queryMods...)
	return &TableQuery[T, Tslice, Topt]{*vq}
}
