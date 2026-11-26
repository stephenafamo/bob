package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"slices"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/dm"
	"github.com/stephenafamo/bob/dialect/mysql/im"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/um"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/internal/ormruntime"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type (
	setter[T any]                = orm.Setter[T, dialect.InsertEditor, dialect.UpdateEditor]
	rowOperations[T any, S ~[]T] = ormruntime.Rows[T, S, *ViewQuery[T, S]]
)

func NewTable[T any, Tset setter[T], C bob.Expression](
	tableName string, columns C, rows orm.RowMapping[T, []T], overwrite func(Tset, T), uniques ...[]string,
) *Table[T, []T, Tset, C] {
	return NewTablex[T, []T, Tset](tableName, columns, nil, rows, overwrite, uniques...)
}

// NewTablex creates a Table with a row mapping and a custom scanner.
// After a successful single-row update, overwrite applies the setter to the original row.
// If scanner is nil, it falls back to [scan.StructMapper].
func NewTablex[T any, Tslice ~[]T, Tset setter[T], C bob.Expression](
	tableName string,
	columns C,
	scanner scan.Mapper[T],
	rows orm.RowMapping[T, Tslice],
	overwrite func(Tset, T),
	uniques ...[]string,
) *Table[T, Tslice, Tset, C] {
	view := NewViewx[T, Tslice](tableName, columns, scanner)
	write := mappings.WriteColumns[T](scanner != nil, view.ColumnsExpr().Names())
	t := &Table[T, Tslice, Tset, C]{
		View:             view,
		rowOperations:    ormruntime.NewRows(rows, view.lookup),
		overwrite:        overwrite,
		nonGeneratedCols: write.NonGenerated,
		uniqueIdx:        uniqueIndexes(write.NonGenerated, uniques...),
	}

	if len(write.AutoIncrement) == 1 {
		t.autoIncrementColumn = write.AutoIncrement[0]
	}

	return t
}

// The table contains extract information from the struct and contains
// caches ???
type Table[T any, Tslice ~[]T, Tset setter[T], C bob.Expression] struct {
	*View[T, Tslice, C]
	rowOperations[T, Tslice]
	overwrite        func(Tset, T)
	nonGeneratedCols []string

	BeforeInsertHooks bob.Hooks[Tset, bob.SkipModelHooksKey]
	AfterInsertHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeUpdateHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterUpdateHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	BeforeDeleteHooks bob.Hooks[Tslice, bob.SkipModelHooksKey]
	AfterDeleteHooks  bob.Hooks[Tslice, bob.SkipModelHooksKey]

	InsertQueryHooks bob.QueryHooks[InsertEditor]
	UpdateQueryHooks bob.QueryHooks[UpdateEditor]
	DeleteQueryHooks bob.QueryHooks[DeleteEditor]

	// The AUTO_INCREMENT column that we can use to retrieve values using lastInsertID
	// If empty, there is no auto inc
	autoIncrementColumn string

	// field indexes of unique columns
	uniqueIdx [][]int
}

// Starts an insert query for this table
func (t *Table[T, Tslice, Tset, C]) Insert(queryMods ...bob.Mod[dialect.InsertEditor]) *insertQuery[T, Tslice, Tset, C] {
	return &insertQuery[T, Tslice, Tset, C]{
		ExecQuery: orm.ExecQuery[dialect.InsertEditor]{
			QueryBuilder: dialect.NewInsert(t.InsertQueryHooks.GetHooks(), bob.ModFunc[dialect.InsertEditor](func(ast dialect.InsertEditor) {
				im.Into(t.NameExpr(), t.nonGeneratedCols...).Edit(ast)
				bob.Mods[dialect.InsertEditor](queryMods).Edit(ast)
			})),
		},
		table: t,
	}
}

// Starts an update query for this table
func (t *Table[T, Tslice, Tset, C]) Update(queryMods ...bob.Mod[dialect.UpdateEditor]) *orm.ExecQuery[dialect.UpdateEditor] {
	return &orm.ExecQuery[dialect.UpdateEditor]{
		QueryBuilder: dialect.NewUpdate(t.UpdateQueryHooks.GetHooks(), bob.ModFunc[dialect.UpdateEditor](func(ast dialect.UpdateEditor) {
			um.Table(t.NameAsExpr()).Edit(ast)
			bob.Mods[dialect.UpdateEditor](queryMods).Edit(ast)
		})),
	}
}

// Starts a delete query for this table
func (t *Table[T, Tslice, Tset, C]) Delete(queryMods ...bob.Mod[dialect.DeleteEditor]) *orm.ExecQuery[dialect.DeleteEditor] {
	return &orm.ExecQuery[dialect.DeleteEditor]{
		QueryBuilder: dialect.NewDelete(t.DeleteQueryHooks.GetHooks(), bob.ModFunc[dialect.DeleteEditor](func(ast dialect.DeleteEditor) {
			dm.From(t.NameAsExpr()).Edit(ast)
			bob.Mods[dialect.DeleteEditor](queryMods).Edit(ast)
		})),
	}
}

type insertQuery[T any, Ts ~[]T, Tset setter[T], C bob.Expression] struct {
	orm.ExecQuery[dialect.InsertEditor]
	table *Table[T, Ts, Tset, C]
}

func (t insertQuery[T, Ts, Tset, C]) WithMods(mods ...bob.Mod[dialect.InsertEditor]) *insertQuery[T, Ts, Tset, C] {
	t.QueryBuilder = t.QueryBuilder.WithMods(mods...)
	return &t
}

// Insert One Row
// NOTE: Because MySQL does not support RETURNING, this will insert the row and then run a SELECT query
// to retrieve the row.
// if there is no AUTO_INCREMENT column and the row was not inserted with unique values, it will return [orm.ErrCannotRetrieveRow]
// [orm.ErrCannotRetrieveRow] is also returned if its a query of the form INSERT INTO ... SELECT ...
func (t *insertQuery[T, Ts, Tset, C]) One(ctx context.Context, exec bob.Executor) (T, error) {
	ctx, q, err := t.insertAll(ctx, exec)
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
func (t *insertQuery[T, Ts, Tset, C]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	ctx, q, err := t.insertAll(ctx, exec)
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
func (t *insertQuery[T, Ts, Tset, C]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	ctx, q, err := t.insertAll(ctx, exec)
	if err != nil {
		return nil, err
	}

	return bob.Cursor(ctx, exec, q, t.table.scanner)
}

type insertBatch struct {
	first   InsertQuery
	columns []string
	rows    []clause.Value
}

func (t *Table[T, Tslice, Tset, C]) retrievable(ast dialect.InsertEditor) error {
	if ast.HasInsertQuery() {
		return fmt.Errorf("inserting from query: %w", orm.ErrCannotRetrieveRow)
	}

	if ast.HasDuplicateKeyUpdate() {
		return fmt.Errorf("has duplicate key update: %w", orm.ErrCannotRetrieveRow)
	}

	if t.autoIncrementColumn != "" {
		return nil
	}

	if ast.HasAssignments() {
		return fmt.Errorf("INSERT SET readback requires an auto increment column: %w", orm.ErrCannotRetrieveRow)
	}

	if len(t.uniqueIdx) > 0 {
		return nil
	}

	return fmt.Errorf("no auto increment column or unique index: %w", orm.ErrCannotRetrieveRow)
}

// insertAll returns the hook context for the SELECT and its loaders.
func (t *insertQuery[T, Ts, Tset, C]) insertAll(ctx context.Context, exec bob.Executor) (context.Context, bob.Query, error) {
	ctx, q, err := ormruntime.ResolveQueryBuilder(ctx, exec, t.QueryBuilder)
	if err != nil {
		return ctx, nil, err
	}
	// Resolve the entire batch before splitting it. Both the write and the
	// lookup must see the same columns, rows, and contextual-mod results.
	var batch insertBatch
	var n int
	batch.first = q.WithMods(bob.ModFunc[dialect.InsertEditor](func(ast dialect.InsertEditor) {
		ctx, err = ast.ResolveContextualMods(ctx)
		if err != nil {
			return
		}
		if err = t.table.retrievable(ast); err != nil {
			return
		}
		batch.columns, batch.rows = ast.ColumnNames(), ast.Rows()
		n = len(batch.rows)
		if ast.HasAssignments() {
			n = 1 // INSERT ... SET always writes one row.
		} else if n > 1 {
			ast.SetRows(batch.rows[0])
		}
	}))
	if err != nil {
		return ctx, nil, err
	}
	if n == 0 {
		return ctx, nil, fmt.Errorf("no insert rows: %w", orm.ErrCannotRetrieveRow)
	}

	results := make([]sql.Result, n)
	for i := range n {
		stmt := batch.first
		if i > 0 {
			stmt = stmt.WithMods(bob.ModFunc[dialect.InsertEditor](func(ast dialect.InsertEditor) {
				ast.SetRows(batch.rows[i])
			}))
		}
		// Hooks have run; the readback SELECT owns the scan and load callbacks.
		query, args, err := stmt.Build(ctx, exec)
		if err != nil {
			return ctx, nil, err
		}
		results[i], err = exec.ExecContext(ctx, query, args...)
		if err != nil {
			return ctx, nil, err
		}
	}

	readback, err := t.table.getInserted(ctx, batch, results)
	return ctx, readback, err
}

func (t *Table[T, Tslice, Tset, C]) getInserted(
	ctx context.Context,
	batch insertBatch,
	results []sql.Result,
) (bob.Query, error) {
	w := &bytes.Buffer{}

	var autoIncrArgs []bob.Expression
	idArgs := make([][]bob.Expression, len(t.uniqueIdx))

	if t.autoIncrementColumn != "" {
		for _, result := range results {
			lastID, err := result.LastInsertId()
			if err != nil {
				return nil, err
			}

			autoIncrArgs = append(autoIncrArgs, Arg(lastID))
		}
	} else {
		for _, val := range batch.rows {
			uIdx, uArgs := t.uniqueSet(ctx, w, batch.columns, val)
			if uIdx == -1 || len(uArgs) == 0 {
				return nil, fmt.Errorf("no unique index found: %w", orm.ErrCannotRetrieveRow)
			}

			idArgs[uIdx] = append(idArgs[uIdx], Group(uArgs...))
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
			filters = append(filters, Quote(t.nonGeneratedCols[uCols[0]]).In(args...))
			continue
		}

		filters = append(filters, Group(t.uniqueColNames(i)...).In(args...))
	}

	return bob.QueryFromExpression(Select(
		sm.Columns(t.projection), sm.From(t.NameAsExpr()), sm.Where(Or(filters...)),
		bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
			q.SetLoaders(batch.first.GetLoaders(ctx)...)
			q.SetMapperMods(batch.first.GetMapperMods(ctx)...)
		}),
	), dialect.Dialect, bob.QueryTypeInsert), nil
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

func isDefaultOrNull(ctx context.Context, w *bytes.Buffer, e bob.Expression) bool {
	w.Reset()

	if e == nil {
		return true
	}

	args, err := e.WriteSQL(ctx, w, dialect.Dialect, 1)
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

func (t *Table[T, Tslice, Tset, C]) uniqueSet(ctx context.Context, w *bytes.Buffer, columns []string, row []bob.Expression) (int, []bob.Expression) {
Outer:
	for whichUnique, unique := range t.uniqueIdx {
		colVals := make([]bob.Expression, 0, len(unique))

		for _, uniqueCol := range unique {
			insertColIndex := slices.Index(columns, t.nonGeneratedCols[uniqueCol])
			if insertColIndex < 0 || insertColIndex >= len(row) {
				continue Outer
			}
			insertedField := row[insertColIndex]

			if insertedField == nil || isDefaultOrNull(ctx, w, insertedField) {
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

func (t *Table[T, Tslice, Tset, C]) uniqueColNames(i int) []bob.Expression {
	if i < 0 || i >= len(t.uniqueIdx) {
		return nil
	}

	u := t.uniqueIdx[i]
	colNames := make([]bob.Expression, len(u))

	for i, col := range u {
		colNames[i] = Quote(t.nonGeneratedCols[col])
	}

	return colNames
}
