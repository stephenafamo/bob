package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"reflect"
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

type setter[T any] interface {
	orm.Setter[T, *dialect.InsertQuery, *dialect.UpdateQuery]
}

func NewTable[T orm.Model, Tset setter[T]](tableName string, uniques ...[]string) *Table[T, []T, Tset] {
	return NewTablex[T, []T, Tset](tableName, uniques...)
}

func NewTablex[T orm.Model, Tslice ~[]T, Tset setter[T]](tableName string, uniques ...[]string) *Table[T, Tslice, Tset] {
	var zeroSet Tset

	setMapping := mappings.GetMappings(reflect.TypeOf(zeroSet))

	view, mappings := newView[T, Tslice](tableName)
	t := &Table[T, Tslice, Tset]{
		View:             view,
		setterMapping:    setMapping,
		nonGeneratedCols: internal.FilterNonZero(mappings.NonGenerated),
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
type Table[T orm.Model, Tslice ~[]T, Tset setter[T]] struct {
	*View[T, Tslice]
	pkExpr           dialect.Expression
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

	// save if we can retrieve or not
	unretrievable bool
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset]) Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) *insertQuery[T, Tslice] {
	q := &insertQuery[T, Tslice]{
		ExecQuery: orm.ExecQuery[*dialect.InsertQuery]{
			BaseQuery: Insert(im.Into(t.Name(), t.nonGeneratedCols...)),
			Hooks:     &t.InsertQueryHooks,
		},
		scanner:       t.scanner,
		getInserted:   t.getInserted,
		unretrievable: t.unretrievable,
		hooks:         &t.AfterInsertHooks,
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

type insertQuery[T orm.Model, Ts ~[]T] struct {
	orm.ExecQuery[*dialect.InsertQuery]
	scanner       scan.Mapper[T]
	unretrievable bool
	getInserted   func([]clause.Value, []sql.Result) (bob.Query, error)
	hooks         *bob.Hooks[Ts, bob.SkipModelHooksKey]
}

// Insert One Row
// NOTE: Because MySQL does not support RETURNING, this will insert the row and then run a SELECT query
// to retrieve the row.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts]) One(ctx context.Context, exec bob.Executor) (T, error) {
	q, err := t.insertAll(ctx, exec)
	if err != nil {
		return *new(T), err
	}

	return bob.One(ctx, exec, q, t.scanner)
}

// Insert Many
// NOTE: Because MySQL does not support RETURNING, this will insert EACH ROW with individual queries
// and then attempt to retrieve all the rows using a SELECT query.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	q, err := t.insertAll(ctx, exec)
	if err != nil {
		return nil, err
	}

	return bob.Allx[T, Ts](ctx, exec, q, t.scanner)
}

// Insert Many and return a cursor
// NOTE: Because MySQL does not support RETURNING, this will insert EACH ROW with individual queries
// and then attempt to retrieve all the rows using a SELECT query.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	q, err := t.insertAll(ctx, exec)
	if err != nil {
		return nil, err
	}

	return bob.Cursor(ctx, exec, q, t.scanner)
}

// inserts all and returns the select query
func (t *insertQuery[T, Ts]) insertAll(ctx context.Context, exec bob.Executor) (bob.Query, error) {
	// If unretrievable, we can't retrieve the rows
	// simply execute the query and return
	if t.unretrievable {
		if _, err := t.Exec(ctx, exec); err != nil {
			return nil, err
		}

		return nil, orm.ErrCannotRetrieveRow
	}

	if t.Expression.Values.Query != nil {
		return nil, orm.ErrCannotRetrieveRow
	}

	if len(t.Expression.Values.Vals) == 0 {
		return nil, orm.ErrCannotRetrieveRow
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

func (t *Table[T, Tslice, Tset]) getInserted(vals []clause.Value, results []sql.Result) (bob.Query, error) {
	w := &bytes.Buffer{}

	if t.unretrievable {
		return nil, orm.ErrCannotRetrieveRow
	}

	query := Select(sm.From(t.NameAs()))

	// Change query type to Insert so that the correct hooks are run
	query.QueryType = bob.QueryTypeInsert

	var autoIncrArgs []bob.Expression
	idArgs := make([][]bob.Expression, len(t.uniqueIdx))

	for i, val := range vals {
		if t.autoIncrementColumn != "" {
			lastID, err := results[i].LastInsertId()
			if err != nil {
				return nil, err
			}

			autoIncrArgs = append(autoIncrArgs, Arg(lastID))
		} else {
			uIdx, uArgs := t.uniqueSet(w, val)
			if uIdx == -1 || len(uArgs) == 0 {
				return nil, orm.ErrCannotRetrieveRow
			}

			idArgs[uIdx] = append(idArgs[uIdx], ArgGroup(internal.ToAnySlice(uArgs)...))
		}
	}

	filters := make([]bob.Expression, 0, len(t.uniqueIdx))
	if len(autoIncrArgs) > 0 {
		filters = append(filters, Quote(t.autoIncrementColumn).In(autoIncrArgs...))
	}

	for i, args := range idArgs {
		if len(args) == 0 {
			continue
		}

		uCols := t.uniqueIdx[i]
		if len(uCols) == 1 {
			filters = append(filters, Quote(t.setterMapping.All[uCols[0]]).In(args...))
			continue
		}

		filters = append(filters, Group(t.uniqueColNames(i)...).In(args...))
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

	args, err := e.WriteSQL(context.Background(), w, dialect.Dialect, 1)
	if err != nil {
		return false
	}

	if len(args) > 0 {
		return false
	}

	s := w.String()

	return strings.EqualFold(s, "DEFAULT") || strings.EqualFold(s, "NULL")
}

func (t *Table[T, Tslice, Tset]) uniqueSet(w *bytes.Buffer, row []bob.Expression) (int, []bob.Expression) {
	if len(row) != len(t.nonGeneratedCols) {
		return -1, nil
	}

Outer:
	for i, u := range t.uniqueIdx {
		colVals := make([]bob.Expression, 0, len(u))

		for _, col := range u {
			field := row[col]

			if field == nil || isDefaultOrNull(w, field) {
				continue Outer
			}

			colVals = append(colVals, field)
		}

		if len(colVals) == len(u) {
			return i, colVals
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
