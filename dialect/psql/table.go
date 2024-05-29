package psql

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type setter[T any] interface {
	orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]
}

func NewTable[T orm.Table, Tset setter[T]](schema, tableName string) *Table[T, []T, Tset] {
	return NewTablex[T, []T, Tset](schema, tableName)
}

func NewTablex[T orm.Table, Tslice ~[]T, Tset setter[T]](schema, tableName string) *Table[T, Tslice, Tset] {
	var zeroSet Tset

	setMapping := mappings.GetMappings(reflect.TypeOf(zeroSet))
	view, mappings := newView[T, Tslice](schema, tableName)
	t := &Table[T, Tslice, Tset]{
		View:       view,
		pkCols:     internal.FilterNonZero(mappings.PKs),
		setMapping: setMapping,
	}

	if len(t.pkCols) == 1 {
		t.pkExpr = Quote(t.pkCols[0])
	} else {
		expr := make([]bob.Expression, len(t.pkCols))
		for i, col := range t.pkCols {
			expr[i] = Quote(col)
		}
		t.pkExpr = Group(expr...)
	}

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T orm.Table, Tslice ~[]T, Tset setter[T]] struct {
	*View[T, Tslice]
	pkCols     []string
	pkExpr     dialect.Expression
	setMapping mappings.Mapping

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
	slice, err := t.InsertMany(ctx, exec, row)
	if err != nil {
		return *new(T), err
	}

	if len(slice) == 0 {
		return *new(T), sql.ErrNoRows
	}

	return slice[0], nil
}

// InsertMany inserts rows into the table with only the set columns in Tset
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
		im.Returning(t.Columns()),
	)

	for _, row := range rows {
		row.InsertMod().Apply(q.Expression)
	}

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
func (t *Table[T, Tslice, Tset]) Update(ctx context.Context, exec bob.Executor, vals Tset, rows ...T) error {
	if len(rows) == 0 {
		return nil
	}

	_, err := t.BeforeUpdateHooks.Do(ctx, exec, rows)
	if err != nil {
		return err
	}

	pkPairs := make([]bob.Expression, len(rows))
	for i, row := range rows {
		pkPairs[i] = row.PrimaryKeyVals()
	}

	q := Update(um.Table(t.NameAs(ctx)), vals, um.Where(t.pkExpr.In(pkPairs...)))

	ctx, err = t.UpdateQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return err
	}

	if _, err = q.Exec(ctx, exec); err != nil {
		return err
	}

	for _, row := range rows {
		vals.Overwrite(row)
	}

	if _, err = t.AfterUpdateHooks.Do(ctx, exec, rows); err != nil {
		return err
	}

	return nil
}

// Uses the setional columns to know what to insert
// If conflictCols is nil, it uses the primary key columns
// If updateCols is nil, it updates all the columns set in Tset
// if no column is set in Tset (i.e. INSERT DEFAULT VALUES), then it upserts all NonPK columns
func (t *Table[T, Tslice, Tset]) Upsert(ctx context.Context, exec bob.Executor, updateOnConflict bool, conflictCols, updateCols []string, row Tset) (T, error) {
	slice, err := t.UpsertMany(ctx, exec, updateOnConflict, conflictCols, updateCols, row)
	if err != nil {
		return *new(T), err
	}

	if len(slice) == 0 {
		return *new(T), sql.ErrNoRows
	}

	return slice[0], nil
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

	// Just get the set columns in the first row
	columns := rows[0].SetColumns()

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
			DoUpdate(im.SetExcluded(excludeSetCols...))
	}

	q := Insert(
		im.Into(t.NameAs(ctx), internal.FilterNonZero(t.setMapping.NonGenerated)...),
		im.Returning(t.Columns()),
		conflictQM,
	)

	for _, row := range rows {
		row.InsertMod().Apply(q.Expression)
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
func (t *Table[T, Tslice, Tset]) Delete(ctx context.Context, exec bob.Executor, rows ...T) error {
	if len(rows) == 0 {
		return nil
	}

	_, err := t.BeforeDeleteHooks.Do(ctx, exec, rows)
	if err != nil {
		return err
	}

	pkPairs := make([]bob.Expression, len(rows))
	for i, row := range rows {
		pkPairs[i] = row.PrimaryKeyVals()
	}

	q := Delete(dm.From(t.NameAs(ctx)), dm.Where(t.pkExpr.In(pkPairs...)))

	ctx, err = t.DeleteQueryHooks.Do(ctx, exec, q.Expression)
	if err != nil {
		return err
	}

	if _, err = q.Exec(ctx, exec); err != nil {
		return err
	}

	if _, err = t.AfterDeleteHooks.Do(ctx, exec, rows); err != nil {
		return err
	}

	return nil
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset]) InsertQ(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.InsertQuery]) *TableQuery[*dialect.InsertQuery, T, Tslice] {
	q := &TableQuery[*dialect.InsertQuery, T, Tslice]{
		BaseQuery: Insert(im.Into(t.NameAs(ctx), internal.FilterNonZero(t.setMapping.NonGenerated)...)),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.InsertQueryHooks,
	}

	q.Apply(queryMods...)

	return q
}

// Starts an update query for this table
func (t *Table[T, Tslice, Tset]) UpdateQ(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.UpdateQuery]) *TableQuery[*dialect.UpdateQuery, T, Tslice] {
	q := &TableQuery[*dialect.UpdateQuery, T, Tslice]{
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
func (t *Table[T, Tslice, Tset]) DeleteQ(ctx context.Context, exec bob.Executor, queryMods ...bob.Mod[*dialect.DeleteQuery]) *TableQuery[*dialect.DeleteQuery, T, Tslice] {
	q := &TableQuery[*dialect.DeleteQuery, T, Tslice]{
		BaseQuery: Delete(dm.From(t.NameAs(ctx))),
		ctx:       ctx,
		exec:      exec,
		view:      t.View,
		hooks:     &t.DeleteQueryHooks,
	}

	q.Apply(queryMods...)

	return q
}

type returnable interface {
	bob.Expression
	HasReturning() bool
	AppendReturning(...any)
}

type TableQuery[Q returnable, T any, Ts ~[]T] struct {
	bob.BaseQuery[Q]
	ctx   context.Context
	exec  bob.Executor
	view  *View[T, Ts]
	hooks *orm.Hooks[Q, orm.SkipQueryHooksKey]
}

func (t *TableQuery[Q, T, Ts]) hook() error {
	var err error
	t.ctx, err = t.hooks.Do(t.ctx, t.exec, t.Expression)
	return err
}

func (t *TableQuery[Q, T, Ts]) addReturning() {
	if !t.BaseQuery.Expression.HasReturning() {
		t.BaseQuery.Expression.AppendReturning(t.view.Columns())
	}
}

func (t *TableQuery[Q, T, Ts]) afterSelect(exec bob.Executor) bob.ExecOption[T] {
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
func (t *TableQuery[Q, T, Tslice]) Exec() (int64, error) {
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
func (t *TableQuery[Q, T, Tslice]) One() (T, error) {
	t.addReturning()
	if err := t.hook(); err != nil {
		return *new(T), err
	}
	return bob.One(t.ctx, t.exec, t, t.view.scanner, t.afterSelect(t.exec))
}

// All matching rows
func (t *TableQuery[Q, T, Tslice]) All() (Tslice, error) {
	t.addReturning()
	if err := t.hook(); err != nil {
		return nil, err
	}
	return bob.Allx[T, Tslice](t.ctx, t.exec, t, t.view.scanner, t.afterSelect(t.exec))
}

// Cursor to scan through the results
func (t *TableQuery[Q, T, Tslice]) Cursor() (scan.ICursor[T], error) {
	t.addReturning()
	if err := t.hook(); err != nil {
		return nil, err
	}
	return bob.Cursor(t.ctx, t.exec, t, t.view.scanner, t.afterSelect(t.exec))
}
