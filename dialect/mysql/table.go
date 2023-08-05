package mysql

import (
	"context"
	"database/sql"
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

func NewTable[T orm.Table, Tset orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]](tableName string, uniques ...[]string) *Table[T, []T, Tset] {
	return NewTablex[T, []T, Tset](tableName, uniques...)
}

func NewTablex[T orm.Table, Tslice ~[]T, Tset orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]](tableName string, uniques ...[]string) *Table[T, Tslice, Tset] {
	var zeroSet Tset

	setMapping := internal.GetMappings(reflect.TypeOf(zeroSet))

	view, mappings := newView[T, Tslice](tableName)
	t := &Table[T, Tslice, Tset]{
		View:       view,
		setMapping: setMapping,
	}

	pkCols := internal.FilterNonZero(mappings.PKs)
	if len(pkCols) == 1 {
		t.pkExpr = Quote(pkCols[0])
	} else {
		expr := make([]bob.Expression, len(pkCols))
		for i, col := range pkCols {
			expr[i] = Quote(col)
		}
		t.pkExpr = Group(expr...)
	}

	allAutoIncr := internal.FilterNonZero(mappings.AutoIncrement)
	if len(allAutoIncr) == 1 {
		t.autoIncrementColumn = allAutoIncr[0]
	}

	// Do this only if needed
	if t.autoIncrementColumn == "" {
		t.uniqueIdx = uniqueIndexes(setMapping.All, uniques...)
	}

	t.unretrievable = t.autoIncrementColumn == "" && len(t.uniqueIdx) == 0

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T orm.Table, Tslice ~[]T, Tset orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]] struct {
	*View[T, Tslice]
	pkExpr     dialect.Expression
	setMapping internal.Mapping

	BeforeInsertHooks orm.Hooks[[]Tset, orm.SkipModelHooksKey]
	AfterInsertHooks  orm.Hooks[Tslice, orm.SkipModelHooksKey]

	BeforeUpsertHooks orm.Hooks[[]Tset, orm.SkipModelHooksKey]
	AfterUpsertHooks  orm.Hooks[Tslice, orm.SkipModelHooksKey]

	BeforeUpdateHooks orm.Hooks[Tslice, orm.SkipModelHooksKey]
	AfterUpdateHooks  orm.Hooks[Tslice, orm.SkipModelHooksKey]

	BeforeDeleteHooks orm.Hooks[Tslice, orm.SkipModelHooksKey]
	AfterDeleteHooks  orm.Hooks[Tslice, orm.SkipModelHooksKey]

	InsertQueryHooks orm.Hooks[*dialect.InsertQuery, orm.SkipQueryHooksKey]
	UpdateQueryHooks orm.Hooks[*dialect.UpdateQuery, orm.SkipQueryHooksKey]
	DeleteQueryHooks orm.Hooks[*dialect.DeleteQuery, orm.SkipQueryHooksKey]

	// The AUTO_INCREMENT column that we can use to retrieve values using lastInsertID
	// If empty, there is no auto inc
	autoIncrementColumn string

	// field indexes of unique columns
	uniqueIdx [][]int

	// save if we can retrieve or not
	unretrievable bool
}

func (t *Table[T, Tslice, Tset]) getInserted(ctx context.Context, exec bob.Executor, row Tset, result sql.Result) (T, error) {
	var zero T

	if t.unretrievable {
		return zero, orm.ErrCannotRetrieveRow
	}

	q2 := t.Query(ctx, exec)
	if t.autoIncrementColumn != "" {
		lastID, err := result.LastInsertId()
		if err != nil {
			return zero, err
		}

		sm.Where(Quote(t.autoIncrementColumn).EQ(Arg(lastID))).Apply(q2.Expression)
	} else {
		uCols, uArgs := t.uniqueSet(row)
		if len(uCols) == 0 {
			return zero, orm.ErrCannotRetrieveRow
		}

		q2 = t.Query(ctx, exec)
		for i := range uCols {
			sm.Where(Quote(uCols[i]).EQ(Arg(uArgs[i]))).Apply(q2.Expression)
		}
	}

	val, err := q2.One()
	if err != nil {
		return zero, err
	}

	return val, nil
}

// Insert inserts a row into the table with only the set columns in Tset
func (t *Table[T, Tslice, Tset]) Insert(ctx context.Context, exec bob.Executor, row Tset) (T, error) {
	slice, err := t.InsertMany(ctx, exec, row)
	if err != nil {
		return *new(T), err
	}

	return slice[0], nil
}

// InsertMany inserts rows into the table with only the set columns in Tset
// NOTE: Because of the lack of support for RETURNING in MySQL, each row is inserted in a separate query
func (t *Table[T, Tslice, Tset]) InsertMany(ctx context.Context, exec bob.Executor, rows ...Tset) (Tslice, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	var err error

	ctx, err = t.BeforeInsertHooks.Do(ctx, exec, rows)
	if err != nil {
		return nil, err
	}

	q := Insert(
		im.Into(t.NameAs(ctx), internal.FilterNonZero(t.setMapping.NonGenerated)...),
	)

	// To prevent unnecessary work, we will do this before we add the rows
	ctx, err = t.InsertQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return nil, err
	}

	if t.unretrievable {
		for _, row := range rows {
			row.Insert().Apply(q.Expression)
		}
		_, err = q.Exec(ctx, exec)
		if err != nil {
			return nil, err
		}

		return nil, orm.ErrCannotRetrieveRow
	}

	inserted := make(Tslice, len(rows))
	for i, row := range rows {
		q.Expression.Values.Vals = nil
		row.Insert().Apply(q.Expression)
		result, err := q.Exec(ctx, exec)
		if err != nil {
			return nil, err
		}
		inserted[i], err = t.getInserted(ctx, exec, rows[i], result)
		if err != nil {
			return nil, err
		}
	}

	_, err = t.AfterInsertHooks.Do(ctx, exec, inserted)
	if err != nil {
		return nil, err
	}

	return inserted, nil
}

// Updates the given model
// if columns is nil, every non-primary-key column is updated
// NOTE: values from the DB are not refreshed into the model
func (t *Table[T, Tslice, Tset]) Update(ctx context.Context, exec bob.Executor, vals Tset, rows ...T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	_, err := t.BeforeUpdateHooks.Do(ctx, exec, rows)
	if err != nil {
		return 0, err
	}

	pkPairs := make([]bob.Expression, len(rows))
	for i, row := range rows {
		pkPairs[i] = row.PrimaryKeyVals()
	}

	q := Update(um.Table(t.NameAs(ctx)), vals, um.Where(t.pkExpr.In(pkPairs...)))

	ctx, err = t.UpdateQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return 0, err
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	for _, row := range rows {
		vals.Overwrite(row)
	}

	_, err = t.AfterUpdateHooks.Do(ctx, exec, rows)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Uses the setional columns to know what to insert
// If updateCols is nil, it updates all the columns set in Tset
// NOTE: Because of the lack of support for RETURNING in MySQL, each row is inserted in a separate query
func (t *Table[T, Tslice, Tset]) Upsert(ctx context.Context, exec bob.Executor, updateOnConflict bool, updateCols []string, row Tset) (T, error) {
	slice, err := t.UpsertMany(ctx, exec, updateOnConflict, updateCols, row)
	if err != nil {
		return *new(T), err
	}

	return slice[0], nil
}

// Uses the setional columns to know what to insert
// If updateCols is nil, it updates all the columns set in Tset
// NOTE: Because of the lack of support for RETURNING in MySQL, each row is inserted in a separate query
func (t *Table[T, Tslice, Tset]) UpsertMany(ctx context.Context, exec bob.Executor, updateOnConflict bool, updateCols []string, rows ...Tset) (Tslice, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	var err error

	ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, rows)
	if err != nil {
		return nil, err
	}

	// Just get the set columns in the first row
	columns := rows[0].SetColumns()

	var conflictQM bob.Mod[*dialect.InsertQuery]
	if !updateOnConflict {
		conflictQM = im.Ignore()
	} else {
		if len(updateCols) == 0 {
			updateCols = columns
		}

		conflictQM = im.OnDuplicateKeyUpdate().SetValues(updateCols...)
	}

	q := Insert(
		im.Into(t.Name(ctx), columns...),
		conflictQM,
	)

	// To prevent unnecessary work, we will do this before we add the rows
	ctx, err = t.InsertQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return nil, err
	}

	if t.unretrievable {
		for _, row := range rows {
			row.Insert().Apply(q.Expression)
		}

		_, err = q.Exec(ctx, exec)
		if err != nil {
			return nil, err
		}

		return nil, orm.ErrCannotRetrieveRow
	}

	upserted := make(Tslice, len(rows))
	for i, row := range rows {
		q.Expression.Values.Vals = nil
		row.Insert().Apply(q.Expression)
		result, err := q.Exec(ctx, exec)
		if err != nil {
			return nil, err
		}
		upserted[i], err = t.getInserted(ctx, exec, rows[i], result)
		if err != nil {
			return nil, err
		}
	}

	_, err = t.AfterUpsertHooks.Do(ctx, exec, upserted)
	if err != nil {
		return nil, err
	}

	return upserted, nil
}

// Deletes the given model
// if columns is nil, every column is deleted
func (t *Table[T, Tslice, Tset]) Delete(ctx context.Context, exec bob.Executor, rows ...T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	_, err := t.BeforeDeleteHooks.Do(ctx, exec, rows)
	if err != nil {
		return 0, err
	}

	pkPairs := make([]bob.Expression, len(rows))
	for i, row := range rows {
		pkPairs[i] = row.PrimaryKeyVals()
	}

	q := Delete(dm.From(t.NameAs(ctx)), dm.Where(t.pkExpr.In(pkPairs...)))

	ctx, err = t.DeleteQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return 0, err
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterDeleteHooks.Do(ctx, exec, rows)
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

func (t *Table[T, Tslice, Tset]) uniqueSet(row Tset) ([]string, []any) {
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

			colNames = append(colNames, t.setMapping.All[col])
			args = append(args, field.Interface())
		}

		if len(colNames) == len(u) {
			return colNames, args
		}
	}

	return nil, nil
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset]) InsertQ(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.InsertQuery]) *TQuery[*dialect.InsertQuery, T, Tslice] {
	q := &TQuery[*dialect.InsertQuery, T, Tslice]{
		BaseQuery: Insert(im.Into(t.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.InsertQueryHooks,
	}

	q.Apply(queryMods...)

	return q
}

// Starts an update query for this table
func (t *Table[T, Tslice, Tset]) UpdateQ(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.UpdateQuery]) *TQuery[*dialect.UpdateQuery, T, Tslice] {
	q := &TQuery[*dialect.UpdateQuery, T, Tslice]{
		BaseQuery: Update(um.Table(t.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.UpdateQueryHooks,
	}

	q.Apply(queryMods...)

	return q
}

// Starts a delete query for this table
func (t *Table[T, Tslice, Tset]) DeleteQ(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.DeleteQuery]) *TQuery[*dialect.DeleteQuery, T, Tslice] {
	q := &TQuery[*dialect.DeleteQuery, T, Tslice]{
		BaseQuery: Delete(dm.From(t.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.DeleteQueryHooks,
	}

	q.Apply(queryMods...)

	return q
}

type TQuery[Q bob.Expression, T any, Ts ~[]T] struct {
	bob.BaseQuery[Q]
	ctx   context.Context
	exec  bob.Executor
	view  *View[T, Ts]
	hooks *orm.Hooks[Q, orm.SkipQueryHooksKey]
}

// Execute the query
func (t *TQuery[Q, T, Tslice]) Exec() (int64, error) {
	var err error

	if t.ctx, err = t.hooks.Do(t.ctx, t.exec, t.Expression); err != nil {
		return 0, err
	}

	result, err := t.BaseQuery.Exec(t.ctx, t.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
