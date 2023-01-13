package mysql

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/dm"
	"github.com/stephenafamo/bob/dialect/mysql/im"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/um"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
)

var (
	ErrNothingToUpdate   = errors.New("nothing to update")
	ErrCannotRetrieveRow = errors.New("cannot retrieve inserted row")
)

func NewTable[T any, Tslice ~[]T, Topt any](schema, tableName string, uniques ...[]string) *Table[T, Tslice, Topt] {
	var zeroOpt Topt

	optMapping := internal.GetMappings(reflect.TypeOf(zeroOpt))

	view, mappings := newView[T, Tslice](schema, tableName)
	t := &Table[T, Tslice, Topt]{
		View:       view,
		pkCols:     internal.FilterNonZero(mappings.PKs),
		optMapping: optMapping,
	}

	allAutoIncr := internal.FilterNonZero(mappings.AutoIncrement)
	if len(allAutoIncr) == 1 {
		optAutoIncr := internal.FilterNonZero(optMapping.AutoIncrement)
		if len(allAutoIncr) == len(optAutoIncr) && allAutoIncr[0] == optAutoIncr[0] {
			t.autoIncrementColumn = allAutoIncr[0]
			return t
		}
	}

	// Do this only if needed
	if t.autoIncrementColumn == "" {
		t.uniqueIdx = uniqueIndexes(optMapping.All, uniques...)
	}

	t.unretrievable = t.autoIncrementColumn == "" && len(t.uniqueIdx) == 0

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Topt any] struct {
	*View[T, Tslice]
	pkCols     []string
	optMapping internal.Mapping

	BeforeInsertHooks orm.Hooks[Topt]
	BeforeUpsertHooks orm.Hooks[Topt]

	// NOTE: This is not called by InsertMany()
	AfterInsertOneHooks orm.Hooks[T]
	// NOTE: This is not called by UpsertMany()
	AfterUpsertOneHooks orm.Hooks[T]

	BeforeUpdateHooks orm.Hooks[T]
	AfterUpdateHooks  orm.Hooks[T]

	BeforeDeleteHooks orm.Hooks[T]
	AfterDeleteHooks  orm.Hooks[T]

	// The AUTO_INCREMENT column that we can use to retrieve values using lastInsertID
	// If empty, there is no auto inc
	autoIncrementColumn string

	// field indexes of unique columns
	uniqueIdx [][]int

	// save if we can retrieve or not
	unretrievable bool
}

// Insert inserts a row into the table with only the set columns in Topt
//   - If the table has an AUTO_INCREMENT column,
//     the inserted row is retrieved using the lastInsertID
//   - If there is no AUTO_INCREMENT but the table has a unique indes that
//     has all columns set in the optional row, then the values of the unique columns
//     are used to retrieve the inserted row
//
// If there is none of the above methods are possible, a zero value and
// [ErrCannotRetrieveRow] is returned after a successful insert
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
		im.Into(t.Name(ctx), columns...),
		im.Rows(values...),
	)

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return zero, err
	}

	if t.unretrievable {
		return zero, ErrCannotRetrieveRow
	}

	if t.autoIncrementColumn != "" {
		lastID, err := result.LastInsertId()
		if err != nil {
			return zero, err
		}

		return t.Query(ctx, exec, sm.Where(Quote(t.autoIncrementColumn).EQ(Arg(lastID)))).One()
	}

	uCols, uArgs := t.uniqueSet(row)
	if len(uCols) == 0 {
		return zero, ErrCannotRetrieveRow
	}

	q2 := t.Query(ctx, exec)
	for i := range uCols {
		q2.Apply(sm.Where(Quote(uCols[i]).EQ(Arg(uArgs[i]))))
	}

	val, err := q2.One()
	if err != nil {
		return zero, err
	}

	_, err = t.AfterInsertOneHooks.Do(ctx, exec, val)
	if err != nil {
		return val, err
	}

	return val, nil
}

// InsertMany inserts multiple row into the table with only the set columns in Topt
// and returns the number of inserted rows
func (t *Table[T, Tslice, Topt]) InsertMany(ctx context.Context, exec bob.Executor, rows ...Topt) (int64, error) {
	var err error

	for _, row := range rows {
		ctx, err = t.BeforeInsertHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, nil
		}
	}

	columns, values, err := internal.GetColumnValues(t.optMapping, nil, rows...)
	if err != nil {
		return 0, fmt.Errorf("get insert values: %w", err)
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
		im.Into(t.Name(ctx), columns...),
		im.Rows(values...),
	)

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
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
// If updateCols is nil, it updates all the columns set in Topt
// if no column is set in Topt (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Topt]) Upsert(ctx context.Context, exec bob.Executor, updateOnConflict bool, updateCols []string, row Topt) (T, error) {
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

	var conflictQM bob.Mod[*dialect.InsertQuery]
	if !updateOnConflict {
		conflictQM = im.Ignore()
	} else {
		if len(updateCols) == 0 {
			updateCols = columns
		}

		conflictQM = im.OnDuplicateKeyUpdate().Set(t.alias, updateCols...)
	}

	q := Insert(
		im.Into(t.Name(ctx), columns...),
		im.Rows(values...),
		im.As(t.alias),
		conflictQM,
	)

	result, err := bob.Exec(ctx, exec, q)
	if err != nil {
		return zero, err
	}

	if t.unretrievable {
		return zero, ErrCannotRetrieveRow
	}

	if t.autoIncrementColumn != "" {
		lastID, err := result.LastInsertId()
		if err != nil {
			return zero, err
		}

		return t.Query(ctx, exec, sm.Where(Quote(t.autoIncrementColumn).EQ(Arg(lastID)))).One()
	}

	uCols, uArgs := t.uniqueSet(row)
	if len(uCols) == 0 {
		return zero, ErrCannotRetrieveRow
	}

	q2 := t.Query(ctx, exec)
	for i := range uCols {
		q2.Apply(sm.Where(Quote(uCols[i]).EQ(Arg(uArgs[i]))))
	}

	val, err := q2.One()
	if err != nil {
		return zero, err
	}

	_, err = t.AfterUpsertOneHooks.Do(ctx, exec, val)
	if err != nil {
		return val, err
	}

	return val, nil
}

// Uses the optional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Topt
// if no column is set in Topt (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Topt]) UpsertMany(ctx context.Context, exec bob.Executor, updateOnConflict bool, updateCols []string, rows ...Topt) (int64, error) {
	var err error

	for _, row := range rows {
		ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, row)
		if err != nil {
			return 0, nil
		}
	}

	columns, values, err := internal.GetColumnValues(t.optMapping, updateCols, rows...)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}

	// If there are no columns, force at least one column with "DEFAULT" for each row
	if len(columns) == 0 {
		columns = []string{internal.FirstNonEmpty(t.optMapping.All)}
		values = make([][]any, len(rows))
		for i := range rows {
			values[i] = []any{"DEFAULT"}
		}
	}

	var conflictQM bob.Mod[*dialect.InsertQuery]
	if !updateOnConflict {
		conflictQM = im.Ignore()
	} else {
		if len(updateCols) == 0 {
			updateCols = columns
		}

		conflictQM = im.OnDuplicateKeyUpdate().Set(t.alias, updateCols...)
	}

	q := Insert(
		im.Into(t.Name(ctx), columns...),
		conflictQM,
	)

	for _, val := range values {
		q.Apply(im.Values(val...))
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
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
	t.BaseQuery.Expression.SelectList.Columns = pkGroup
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
func (t *TableQuery[T, Tslice, Topt]) DeleteAll() (int64, error) {
	q := Delete(dm.From(t.nameExpr(t.ctx)))

	pkGroup := make([]any, len(t.pkCols))
	for i, pk := range t.pkCols {
		pkGroup[i] = Quote(pk)
	}

	// Select ONLY the primary keys
	t.BaseQuery.Expression.SelectList.Columns = pkGroup
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

func uniqueIndexes(allCols []string, uniques ...[]string) [][]int {
	var indexes [][]int
	for _, unique := range uniques {
		index := make([]int, 0, len(unique))
		for _, name := range unique {
			for i, col := range allCols {
				if name == col {
					index = append(index, i)
				}
			}
		}

		// all columns found
		if len(index) == len(unique) {
			indexes = append(indexes, index)
		}
	}

	return indexes
}

//nolint:gochecknoglobals
var settableTyp = reflect.TypeOf((*interface{ IsSet() bool })(nil)).Elem()

func (t *Table[T, Tslice, Topt]) uniqueSet(row Topt) ([]string, []any) {
	val := reflect.ValueOf(row)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	for _, u := range t.uniqueIdx {
		colNames := make([]string, 0, len(u))
		args := make([]any, 0, len(u))
		for _, col := range u {
			field := val.Field(col)

			// If it does not implement the type, break
			if !field.Type().Implements(settableTyp) {
				break
			}

			// if it is not set break
			if !field.MethodByName("IsSet").Call(nil)[0].Interface().(bool) {
				break
			}

			colNames = append(colNames, t.optMapping.All[col])
			args = append(args, field.Interface())
		}

		if len(colNames) == len(u) {
			return colNames, args
		}
	}

	return nil, nil
}
