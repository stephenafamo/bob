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
	"github.com/stephenafamo/scan"
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
}

// Insert inserts a row into the table with only the set columns in Tset
func (t *Table[T, Tslice, Tset]) Insert(ctx context.Context, exec bob.Executor, row Tset) (T, error) {
	var err error
	var zero T

	ctx, err = t.BeforeInsertHooks.Do(ctx, exec, []Tset{row})
	if err != nil {
		return zero, err
	}

	columns, values, err := internal.GetColumnValues(t.setMapping.NonGenerated, nil, row)
	if err != nil {
		return zero, fmt.Errorf("get insert values: %w", err)
	}

	q := Insert(
		im.Into(t.NameAs(ctx), columns...),
		im.Rows(values...),
		im.Returning(t.Columns()),
	)

	ctx, err = t.InsertQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return zero, err
	}

	val, err := bob.One(ctx, exec, q, t.scanner)
	if err != nil {
		return val, err
	}

	_, err = t.AfterInsertHooks.Do(ctx, exec, Tslice{val})
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

	ctx, err = t.BeforeInsertHooks.Do(ctx, exec, rows)
	if err != nil {
		return nil, err
	}

	columns, values, err := internal.GetColumnValues(t.setMapping.NonGenerated, nil, rows...)
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

	ctx, err = t.InsertQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return nil, err
	}

	vals, err := bob.All(ctx, exec, q, t.scanner)
	if err != nil {
		return vals, err
	}

	_, err = t.AfterInsertHooks.Do(ctx, exec, vals)
	if err != nil {
		return vals, err
	}

	return vals, nil
}

// Updates the given model
// if columns is nil, every non-primary-key column is updated
// NOTE: values from the DB are not refreshed into the model
func (t *Table[T, Tslice, Tset]) Update(ctx context.Context, exec bob.Executor, row T, cols ...string) (int64, error) {
	_, err := t.BeforeUpdateHooks.Do(ctx, exec, Tslice{row})
	if err != nil {
		return 0, err
	}

	q := Update(um.Table(t.NameAs(ctx)))

	pks, pkVals, err := internal.GetColumnValues(t.mapping.PKs, t.mapping.PKs, row)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	if len(cols) == 0 {
		cols = t.mapping.NonPKs
	}
	columns, values, err := internal.GetColumnValues(t.mapping.NonGenerated, cols, row)
	if err != nil {
		return 0, fmt.Errorf("get update values: %w", err)
	}

	for i, pk := range pks {
		q.Apply(um.Where(Quote(pk).EQ(pkVals[0][i])))
	}

	for i, col := range columns {
		q.Apply(um.Set(col).To(values[0][i]))
	}

	ctx, err = t.UpdateQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return 0, err
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterUpdateHooks.Do(ctx, exec, Tslice{row})
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

	columns, values, err := internal.GetColumnValues(t.setMapping.NonGenerated, nil, vals)
	if err != nil {
		return 0, fmt.Errorf("get upsert values: %w", err)
	}
	if len(columns) == 0 {
		return 0, orm.ErrNothingToUpdate
	}

	_, err = t.BeforeUpdateHooks.Do(ctx, exec, rows)
	if err != nil {
		return 0, err
	}

	q := Update(um.Table(t.NameAs(ctx)))

	for i, col := range columns {
		q.Apply(um.Set(col).To(values[0][i]))
	}

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping.PKs, t.mapping.PKs, rows...)
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

	ctx, err = t.UpdateQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return 0, err
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterUpdateHooks.Do(ctx, exec, rows)
	if err != nil {
		return 0, err
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

	ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, []Tset{row})
	if err != nil {
		return zero, err
	}

	columns, values, err := internal.GetColumnValues(t.setMapping.NonGenerated, nil, row)
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

	ctx, err = t.InsertQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return zero, err
	}

	val, err := bob.One(ctx, exec, q, t.scanner)
	if err != nil {
		return val, err
	}

	_, err = t.AfterUpsertHooks.Do(ctx, exec, Tslice{val})
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

	ctx, err = t.BeforeUpsertHooks.Do(ctx, exec, rows)
	if err != nil {
		return nil, err
	}

	columns, values, err := internal.GetColumnValues(t.setMapping.NonGenerated, nil, rows...)
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

	ctx, err = t.InsertQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return nil, err
	}

	vals, err := bob.All(ctx, exec, q, t.scanner)
	if err != nil {
		return vals, err
	}

	_, err = t.AfterUpsertHooks.Do(ctx, exec, vals)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// Deletes the given model
// if columns is nil, every column is deleted
func (t *Table[T, Tslice, Tset]) Delete(ctx context.Context, exec bob.Executor, row T) (int64, error) {
	_, err := t.BeforeDeleteHooks.Do(ctx, exec, Tslice{row})
	if err != nil {
		return 0, err
	}

	q := Delete(dm.From(t.NameAs(ctx)))

	pks, pkVals, err := internal.GetColumnValues(t.mapping.PKs, t.mapping.PKs, row)
	if err != nil {
		return 0, fmt.Errorf("get update pk values: %w", err)
	}

	for i, pk := range pks {
		q.Apply(dm.Where(Quote(pk).EQ(pkVals[0][i])))
	}

	ctx, err = t.DeleteQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return 0, err
	}

	result, err := q.Exec(ctx, exec)
	if err != nil {
		return 0, err
	}

	_, err = t.AfterDeleteHooks.Do(ctx, exec, Tslice{row})
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

	_, err := t.BeforeDeleteHooks.Do(ctx, exec, rows)
	if err != nil {
		return 0, err
	}

	q := Delete(dm.From(t.NameAs(ctx)))

	// Find a set the PKs
	pks, pkVals, err := internal.GetColumnValues(t.mapping.PKs, t.mapping.PKs, rows...)
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

// Starts an update query for this table
func (t *Table[T, Tslice, Tset]) UpdateAll(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.UpdateQuery]) *TQuery[*dialect.UpdateQuery, T, Tslice] {
	q := &TQuery[*dialect.UpdateQuery, T, Tslice]{
		BaseQuery: Update(um.Table(t.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.UpdateQueryHooks,
	}

	// q.Expression.SetLoadContext(ctx)
	q.Apply(queryMods...)

	return q
}

// Starts a delete query for this table
func (t *Table[T, Tslice, Tset]) DeleteAll(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.DeleteQuery]) *TQuery[*dialect.DeleteQuery, T, Tslice] {
	q := &TQuery[*dialect.DeleteQuery, T, Tslice]{
		BaseQuery: Delete(dm.From(t.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.DeleteQueryHooks,
	}

	// q.Expression.SetLoadContext(ctx)
	q.Apply(queryMods...)

	return q
}

type returnable interface {
	bob.Expression
	HasReturning() bool
	AppendReturning(...any)
}

type TQuery[Q returnable, T any, Ts ~[]T] struct {
	bob.BaseQuery[Q]
	ctx   context.Context
	exec  bob.Executor
	view  *View[T, Ts]
	hooks *orm.Hooks[Q, orm.SkipQueryHooksKey]
}

func (t *TQuery[Q, T, Ts]) hook() error {
	var err error
	t.ctx, err = t.hooks.Do(t.ctx, t.exec, t.Expression)
	return err
}

func (t *TQuery[Q, T, Ts]) addReturning() {
	if t.BaseQuery.Expression.HasReturning() {
		t.BaseQuery.Expression.AppendReturning(t.view.Columns())
	}
}

func (t *TQuery[Q, T, Ts]) afterSelect(ctx context.Context, exec bob.Executor) bob.ExecOption[T] {
	return func(es *bob.ExecSettings[T]) {
		es.AfterSelect = func(ctx context.Context, retrieved []T) error {
			_, err := t.view.AfterSelectHooks.Do(ctx, exec, retrieved)
			if err != nil {
				return err
			}

			return nil
		}
	}
}

// Execute the query
func (t *TQuery[Q, T, Tslice]) Exec() (int64, error) {
	if err := t.hook(); err != nil {
		return 0, err
	}

	result, err := t.BaseQuery.Exec(t.ctx, t.exec)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// First matching row
func (t *TQuery[Q, T, Tslice]) One() (T, error) {
	t.addReturning()
	if err := t.hook(); err != nil {
		return *new(T), err
	}
	return bob.One(t.ctx, t.exec, t, t.view.scanner, t.afterSelect(t.ctx, t.exec))
}

// All matching rows
func (t *TQuery[Q, T, Tslice]) All() (Tslice, error) {
	t.addReturning()
	if err := t.hook(); err != nil {
		return nil, err
	}
	return bob.Allx[T, Tslice](t.ctx, t.exec, t, t.view.scanner, t.afterSelect(t.ctx, t.exec))
}

// Cursor to scan through the results
func (t *TQuery[Q, T, Tslice]) Cursor() (scan.ICursor[T], error) {
	t.addReturning()
	if err := t.hook(); err != nil {
		return nil, err
	}
	return bob.Cursor(t.ctx, t.exec, t, t.view.scanner, t.afterSelect(t.ctx, t.exec))
}
