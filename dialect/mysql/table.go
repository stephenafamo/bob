package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/dm"
	"github.com/stephenafamo/bob/dialect/mysql/im"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/um"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type setter[T any] = orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]

func NewTable[T any, Tset setter[T]](tableName string, uniques ...[]string) *Table[T, []T, Tset] {
	return NewTablex[T, []T, Tset](tableName, uniques...)
}

func NewTablex[T any, Tslice ~[]T, Tset setter[T]](tableName string, uniques ...[]string) *Table[T, Tslice, Tset] {
	setMapping := mappings.GetMappings(reflect.TypeOf(*new(Tset)))
	view, mappings := newView[T, Tslice](tableName)
	t := &Table[T, Tslice, Tset]{
		View:             view,
		pkCols:           orm.NewColumns(mappings.PKs...).WithParent(view.alias),
		setterMapping:    setMapping,
		nonGeneratedCols: internal.FilterNonZero(mappings.NonGenerated),
		uniqueIdx:        uniqueIndexes(setMapping.All, uniques...),
	}

	allAutoIncr := internal.FilterNonZero(mappings.AutoIncrement)
	if len(allAutoIncr) == 1 {
		t.autoIncrementColumn = allAutoIncr[0]
	}

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset setter[T]] struct {
	*View[T, Tslice]
	pkCols           orm.Columns
	setterMapping    mappings.Mapping
	nonGeneratedCols []string

	BeforeInsertHooks bob.Hooks[Tset, bob.SkipModelHooksKey]
	AfterInsertHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeUpdateHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterUpdateHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeDeleteHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterDeleteHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	InsertQueryHooks bob.Hooks[*dialect.InsertQuery, bob.SkipQueryHooksKey]
	UpdateQueryHooks bob.Hooks[*dialect.UpdateQuery, bob.SkipQueryHooksKey]
	DeleteQueryHooks bob.Hooks[*dialect.DeleteQuery, bob.SkipQueryHooksKey]

	// The AUTO_INCREMENT column that we can use to retrieve values using lastInsertID
	// If empty, there is no auto inc
	autoIncrementColumn string

	// field indexes of unique columns
	uniqueIdx [][]int
}

// Returns the primary key columns for this table.
func (t *Table[T, Tslice, Tset]) PrimaryKey() orm.Columns {
	return t.pkCols
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset]) Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) *insertQuery[T, Tslice, Tset] {
	q := &insertQuery[T, Tslice, Tset]{
		ExecQuery: orm.ExecQuery[*dialect.InsertQuery]{
			BaseQuery: Insert(im.Into(t.Name(), t.nonGeneratedCols...)),
			Hooks:     &t.InsertQueryHooks,
		},
		table: t,
	}

	q.Apply(queryMods...)

	return q
}

// Starts an update query for this table
func (t *Table[T, Tslice, Tset]) Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) *orm.ExecQuery[*dialect.UpdateQuery] {
	q := &orm.ExecQuery[*dialect.UpdateQuery]{
		BaseQuery: Update(um.Table(t.NameAs())),
		Hooks:     &t.UpdateQueryHooks,
	}
	q.Apply(queryMods...)

	return q
}

// Starts a delete query for this table
func (t *Table[T, Tslice, Tset]) Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) *orm.ExecQuery[*dialect.DeleteQuery] {
	q := &orm.ExecQuery[*dialect.DeleteQuery]{
		BaseQuery: Delete(dm.From(t.NameAs())),
		Hooks:     &t.DeleteQueryHooks,
	}

	q.Apply(queryMods...)

	return q
}

type insertQuery[T any, Ts ~[]T, Tset setter[T]] struct {
	orm.ExecQuery[*dialect.InsertQuery]
	table *Table[T, Ts, Tset]
}

// Insert One Row
// NOTE: Because MySQL does not support RETURNING, this will insert the row and then run a SELECT query
// to retrieve the row.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts, Tset]) One(ctx context.Context, exec bob.Executor) (T, error) {
	q, err := t.insertAll(ctx, exec)
	if err != nil {
		return *new(T), err
	}

	return bob.One(ctx, exec, q, t.table.scanner)
}

// Insert Many
// NOTE: Because MySQL does not support RETURNING, this will insert EACH ROW with individual queries
// and then attempt to retrieve all the rows using a SELECT query.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts, Tset]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	q, err := t.insertAll(ctx, exec)
	if err != nil {
		return nil, err
	}

	return bob.Allx[bob.SliceTransformer[T, Ts]](ctx, exec, q, t.table.scanner)
}

// Insert Many and return a cursor
// NOTE: Because MySQL does not support RETURNING, this will insert EACH ROW with individual queries
// and then attempt to retrieve all the rows using a SELECT query.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts, Tset]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	q, err := t.insertAll(ctx, exec)
	if err != nil {
		return nil, err
	}

	return bob.Cursor(ctx, exec, q, t.table.scanner)
}

func (t *insertQuery[T, Tslice, Tset]) retrievable() error {
	if t.Expression.Values.Query != nil {
		return fmt.Errorf("inserting from query: %w", orm.ErrCannotRetrieveRow)
	}

	if len(t.Expression.DuplicateKeyUpdate.Set) > 0 {
		return fmt.Errorf("has duplicate key update: %w", orm.ErrCannotRetrieveRow)
	}

	if t.table.autoIncrementColumn != "" {
		return nil
	}

	if len(t.table.uniqueIdx) > 0 {
		return nil
	}

	return fmt.Errorf("no auto increment column or unique index: %w", orm.ErrCannotRetrieveRow)
}

// inserts all and returns the select query
func (t *insertQuery[T, Ts, Tset]) insertAll(ctx context.Context, exec bob.Executor) (bob.Query, error) {
	if retrievalErr := t.retrievable(); retrievalErr != nil {
		return nil, retrievalErr
	}

	var err error

	// Save the existing values
	oldVals := t.Expression.Values.Vals

	// Run hooks
	ctx, err = t.RunHooks(ctx, exec)
	if err != nil {
		return nil, err
	}

	// Clear hooks
	t.Expression.SetHooks()

	results := make([]sql.Result, len(oldVals))
	for i := range oldVals {
		rowVals := oldVals[i : i+1]
		t.Expression.Values.Vals = rowVals

		result, err := bob.Exec(ctx, exec, t.BaseQuery)
		if err != nil {
			return nil, err
		}

		results[i] = result
	}

	// Restore the values
	t.Expression.Values.Vals = oldVals

	return t.getInserted(oldVals, results)
}

func (t *insertQuery[T, Tslice, Tset]) getInserted(vals []clause.Value, results []sql.Result) (bob.Query, error) {
	w := &bytes.Buffer{}

	if retrievalErr := t.retrievable(); retrievalErr != nil {
		return nil, retrievalErr
	}

	query := Select(sm.From(t.table.NameAs()))

	// Change query type to Insert so that the correct hooks are run
	query.QueryType = bob.QueryTypeInsert

	var autoIncrArgs []bob.Expression
	idArgs := make([][]bob.Expression, len(t.table.uniqueIdx))

	for i, val := range vals {
		if t.table.autoIncrementColumn != "" && len(t.Expression.DuplicateKeyUpdate.Set) == 0 {
			lastID, err := results[i].LastInsertId()
			if err != nil {
				return nil, err
			}

			autoIncrArgs = append(autoIncrArgs, Arg(lastID))
		} else {
			uIdx, uArgs := t.uniqueSet(w, val)
			if uIdx == -1 || len(uArgs) == 0 {
				return nil, fmt.Errorf("no unique index found: %w", orm.ErrCannotRetrieveRow)
			}

			idArgs[uIdx] = append(idArgs[uIdx], Group(uArgs...))
		}
	}

	filters := make([]bob.Expression, 0, len(t.table.uniqueIdx))
	if len(autoIncrArgs) > 0 {
		filters = append(filters, Quote(t.table.autoIncrementColumn).In(autoIncrArgs...))
	}

	for i, args := range idArgs {
		if len(args) == 0 {
			continue
		}

		uCols := t.table.uniqueIdx[i]
		if len(uCols) == 1 {
			filters = append(filters, Quote(t.table.setterMapping.All[uCols[0]]).In(args...))
			continue
		}

		filters = append(filters, Group(t.table.uniqueColNames(i)...).In(args...))
	}

	query.Apply(sm.Where(Or(filters...)))

	return query, nil
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

func isDefaultOrNull(w *bytes.Buffer, e bob.Expression) bool {
	w.Reset()

	if e == nil {
		return true
	}

	args, err := e.WriteSQL(context.Background(), w, dialect.Dialect, 1)
	if err != nil {
		return false
	}

	if len(args) > 0 {
		if args[0] == nil {
			return true
		}
		if driverValue, ok := args[0].(driver.Valuer); ok {
			if val, _ := driverValue.Value(); val == nil {
				return true
			}
		}
		return false
	}

	s := w.String()

	return strings.EqualFold(s, "DEFAULT") || strings.EqualFold(s, "NULL")
}

func (t *insertQuery[T, Tslice, Tset]) uniqueSet(w *bytes.Buffer, row []bob.Expression) (int, []bob.Expression) {
Outer:
	for whichUnique, unique := range t.table.uniqueIdx {
		colVals := make([]bob.Expression, 0, len(unique))

		for _, uniqueCol := range unique {
			insertColIndex := slices.Index(t.Expression.Columns, t.table.setterMapping.All[uniqueCol])
			insertedField := row[insertColIndex]

			if insertedField == nil || isDefaultOrNull(w, insertedField) {
				continue Outer
			}

			colVals = append(colVals, insertedField)
		}

		if len(colVals) == len(unique) {
			return whichUnique, colVals
		}
	}

	return -1, nil
}

func (t *Table[T, Tslice, Tset]) uniqueColNames(i int) []bob.Expression {
	if i < 0 || i >= len(t.uniqueIdx) {
		return nil
	}

	u := t.uniqueIdx[i]
	colNames := make([]bob.Expression, len(u))

	for i, col := range u {
		colNames[i] = Quote(t.setterMapping.All[col])
	}

	return colNames
}
