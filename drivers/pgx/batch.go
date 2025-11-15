package pgx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

// BatchBuilder is a helper for building and executing pgx batch operations with Bob queries.
// It wraps pgx.Batch and provides methods to add Bob queries and execute them.
//
// Example usage:
//
//	batch := pgx.NewBatchBuilder()
//	batch.AddQuery(models.Users.Insert(models.SelectWhere.Users.Name.Set("Alice")))
//	batch.AddQuery(models.Users.Insert(models.SelectWhere.Users.Name.Set("Bob")))
//	results := batch.Execute(ctx, tx)
//	defer results.Close()
type BatchBuilder struct {
	batch *pgx.Batch
}

// NewBatchBuilder creates a new BatchBuilder with an empty pgx.Batch.
func NewBatchBuilder() *BatchBuilder {
	return &BatchBuilder{
		batch: &pgx.Batch{},
	}
}

// AddQuery queues a Bob query for batch execution.
// The query is built using bob.Build and the resulting SQL and arguments are added to the batch.
//
// Returns an error if the query cannot be built.
func (b *BatchBuilder) AddQuery(q bob.Query) error {
	sql, args, err := bob.Build(context.Background(), q)
	if err != nil {
		return err
	}
	b.batch.Queue(sql, args...)
	return nil
}

// AddQueryContext is like AddQuery but uses the provided context for building the query.
func (b *BatchBuilder) AddQueryContext(ctx context.Context, q bob.Query) error {
	sql, args, err := bob.Build(ctx, q)
	if err != nil {
		return err
	}
	b.batch.Queue(sql, args...)
	return nil
}

// AddRawQuery queues a raw SQL query with arguments for batch execution.
// This is useful when you need to execute queries that are not built with Bob.
func (b *BatchBuilder) AddRawQuery(sql string, args ...any) {
	b.batch.Queue(sql, args...)
}

// Len returns the number of queries queued in the batch.
func (b *BatchBuilder) Len() int {
	return b.batch.Len()
}

// Execute sends the batch to the database using the provided executor.
// The executor must be a pgx transaction or connection that supports SendBatch.
//
// Returns BatchResults which must be closed after processing all results.
//
// Example:
//
//	results := batch.Execute(ctx, tx)
//	defer results.Close()
func (b *BatchBuilder) Execute(ctx context.Context, exec Executor) BatchResults {
	return NewBatchResults(exec.SendBatch(ctx, b.batch))
}

// Executor interface for types that can send batch operations.
// This is satisfied by pgx.Tx, pgx.Conn, and pgx.Pool types from this package.
type Executor interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// BatchResults wraps pgx.BatchResults with helper methods for processing results.
type BatchResults struct {
	results pgx.BatchResults
}

// NewBatchResults creates a new BatchResults wrapper.
func NewBatchResults(results pgx.BatchResults) BatchResults {
	return BatchResults{results: results}
}

// Close closes the batch results. This must be called after all results are processed.
func (br BatchResults) Close() error {
	return br.results.Close()
}

// Exec processes the next query result that does not return rows.
// This should be used for INSERT, UPDATE, DELETE queries.
func (br BatchResults) Exec() (sql.Result, error) {
	tag, err := br.results.Exec()
	return result{tag}, err
}

// Query processes the next query result that returns rows.
// Returns rows that implement scan.Rows interface compatible with Bob's scanning.
func (br BatchResults) Query() (scan.Rows, error) {
	pgxRows, err := br.results.Query()
	return rows{pgxRows}, err
}

// QueryRow processes the next query result that returns a single row.
func (br BatchResults) QueryRow() pgx.Row {
	return br.results.QueryRow()
}


// BatchHelper provides convenience methods for common batch operations.
type BatchHelper struct {
	ctx  context.Context
	exec Executor
}

// NewBatchHelper creates a new BatchHelper with the given context and executor.
func NewBatchHelper(ctx context.Context, exec Executor) *BatchHelper {
	return &BatchHelper{
		ctx:  ctx,
		exec: exec,
	}
}

// ExecQueries executes multiple queries in a batch and returns the results.
// This is a convenience method for executing queries that don't return rows.
//
// Example:
//
//	helper := pgx.NewBatchHelper(ctx, tx)
//	results, err := helper.ExecQueries(
//	    insertQuery1,
//	    insertQuery2,
//	    updateQuery,
//	)
//
// Returns a slice of sql.Result, one for each query.
func (h *BatchHelper) ExecQueries(queries ...bob.Query) ([]sql.Result, error) {
	batch := NewBatchBuilder()
	for _, q := range queries {
		if err := batch.AddQueryContext(h.ctx, q); err != nil {
			return nil, err
		}
	}

	results := batch.Execute(h.ctx, h.exec)
	defer results.Close()

	var sqlResults []sql.Result
	for i := 0; i < len(queries); i++ {
		res, err := results.Exec()
		if err != nil {
			return sqlResults, err
		}
		sqlResults = append(sqlResults, res)
	}

	return sqlResults, nil
}

// ErrBatchMismatch is returned when the number of queries doesn't match the number of destinations.
var ErrBatchMismatch = errors.New("number of queries must match number of destinations")

// Standalone helper functions for scanning batch results

// ScanOne scans the next query result from BatchResults into a single value.
// This is a convenience function that wraps scan.OneFromRows.
//
// Example:
//
//	user, err := pgx.ScanOne(ctx, results, scan.StructMapper[User]())
func ScanOne[T any](ctx context.Context, br BatchResults, m scan.Mapper[T]) (T, error) {
	pgxRows, err := br.results.Query()
	if err != nil {
		var zero T
		return zero, err
	}
	return scan.OneFromRows(ctx, m, rows{pgxRows})
}

// ScanAll scans all rows from the next query result in BatchResults.
// This is a convenience function that wraps scan.AllFromRows.
//
// Example:
//
//	users, err := pgx.ScanAll(ctx, results, scan.StructMapper[User]())
func ScanAll[T any](ctx context.Context, br BatchResults, m scan.Mapper[T]) ([]T, error) {
	pgxRows, err := br.results.Query()
	if err != nil {
		return nil, err
	}
	return scan.AllFromRows(ctx, m, rows{pgxRows})
}

// ExecRow processes the next query result and validates that exactly one row was affected.
// Returns an error if zero or more than one row was affected.
//
// This is useful for UPDATE/DELETE operations that should affect exactly one row.
//
// Example:
//
//	err := pgx.ExecRow(results)
//	if err != nil {
//		// Either no rows or multiple rows were affected
//	}
func ExecRow(br BatchResults) error {
	tag, err := br.results.Exec()
	if err != nil {
		return err
	}

	rows := tag.RowsAffected()
	if rows == 0 {
		return ErrNoRowsAffected
	}
	if rows > 1 {
		return ErrTooManyRowsAffected
	}

	return nil
}

// ExecRowResult processes the next query result, validates that exactly one row was affected,
// and returns the sql.Result.
//
// Example:
//
//	res, err := pgx.ExecRowResult(results)
func ExecRowResult(br BatchResults) (sql.Result, error) {
	tag, err := br.results.Exec()
	if err != nil {
		return nil, err
	}

	res := result{tag}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return res, ErrNoRowsAffected
	}
	if rows > 1 {
		return res, ErrTooManyRowsAffected
	}

	return res, nil
}

var (
	// ErrNoRowsAffected is returned when ExecRow expects one row but zero were affected
	ErrNoRowsAffected = errors.New("expected 1 row affected, got 0")
	// ErrTooManyRowsAffected is returned when ExecRow expects one row but multiple were affected
	ErrTooManyRowsAffected = errors.New("expected 1 row affected, got multiple")
)

// QueuedBatch provides pgxutil-style batch operations with deferred result population.
// This is an alternative API inspired by pgxutil that allows passing result pointers
// when queuing operations, which are then populated during batch execution.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var user User
//	var count int64
//	pgx.QueueSelectRow(qb, ctx, userQuery, scan.StructMapper[User](), &user)
//	pgx.QueueSelectRow(qb, ctx, countQuery, scan.SingleColumnMapper[int64], &count)
//	err := qb.Execute(ctx, tx)
//	// user and count are now populated
type QueuedBatch struct {
	batch   *pgx.Batch
	actions []batchAction
}

type batchAction func(context.Context, BatchResults) error

// NewQueuedBatch creates a new QueuedBatch.
func NewQueuedBatch() *QueuedBatch {
	return &QueuedBatch{
		batch:   &pgx.Batch{},
		actions: make([]batchAction, 0),
	}
}

// QueueQuery queues a Bob query without expecting a result.
// The result will be validated during Execute but not returned.
func (qb *QueuedBatch) QueueQuery(q bob.Query) error {
	return qb.QueueQueryContext(context.Background(), q)
}

// QueueQueryContext queues a Bob query with context without expecting a result.
func (qb *QueuedBatch) QueueQueryContext(ctx context.Context, q bob.Query) error {
	sql, args, err := bob.Build(ctx, q)
	if err != nil {
		return err
	}

	qb.batch.Queue(sql, args...)
	qb.actions = append(qb.actions, func(execCtx context.Context, br BatchResults) error {
		_, err := br.Exec()
		return err
	})

	return nil
}

// QueueRawQuery queues a raw SQL query without expecting a result.
func (qb *QueuedBatch) QueueRawQuery(sql string, args ...any) {
	qb.batch.Queue(sql, args...)
	qb.actions = append(qb.actions, func(ctx context.Context, br BatchResults) error {
		_, err := br.Exec()
		return err
	})
}

// QueueExecRow queues a query that must affect exactly one row.
// Returns an error during Execute if zero or multiple rows are affected.
func (qb *QueuedBatch) QueueExecRow(q bob.Query) error {
	return qb.QueueExecRowContext(context.Background(), q)
}

// QueueExecRowContext queues a query with context that must affect exactly one row.
func (qb *QueuedBatch) QueueExecRowContext(ctx context.Context, q bob.Query) error {
	sql, args, err := bob.Build(ctx, q)
	if err != nil {
		return err
	}

	qb.batch.Queue(sql, args...)
	qb.actions = append(qb.actions, func(execCtx context.Context, br BatchResults) error {
		return ExecRow(br)
	})

	return nil
}

// QueueUpdateRow queues an UPDATE query that must affect exactly one row.
func (qb *QueuedBatch) QueueUpdateRow(q bob.Query) error {
	return qb.QueueExecRow(q)
}

// QueueUpdateRowContext queues an UPDATE query with context that must affect exactly one row.
func (qb *QueuedBatch) QueueUpdateRowContext(ctx context.Context, q bob.Query) error {
	return qb.QueueExecRowContext(ctx, q)
}

// Generic helper functions for QueuedBatch (standalone functions due to Go's limitation on generic methods)

// QueueSelectRow queues a SELECT query that must return exactly one row.
// The result pointer will be populated during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var user User
//	pgx.QueueSelectRow(qb, ctx, query, scan.StructMapper[User](), &user)
//	err := qb.Execute(ctx, tx)
//	// user is now populated
func QueueSelectRow[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *T) error {
	sql, args, err := bob.Build(ctx, q)
	if err != nil {
		return err
	}

	qb.batch.Queue(sql, args...)
	qb.actions = append(qb.actions, func(execCtx context.Context, br BatchResults) error {
		val, err := ScanOne(execCtx, br, m)
		if err != nil {
			return err
		}
		*result = val
		return nil
	})

	return nil
}

// QueueSelectAll queues a SELECT query that returns multiple rows.
// The result slice pointer will be populated during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var users []User
//	pgx.QueueSelectAll(qb, ctx, query, scan.StructMapper[User](), &users)
//	err := qb.Execute(ctx, tx)
//	// users slice is now populated
func QueueSelectAll[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *[]T) error {
	sql, args, err := bob.Build(ctx, q)
	if err != nil {
		return err
	}

	qb.batch.Queue(sql, args...)
	qb.actions = append(qb.actions, func(execCtx context.Context, br BatchResults) error {
		val, err := ScanAll(execCtx, br, m)
		if err != nil {
			return err
		}
		*result = val
		return nil
	})

	return nil
}

// QueueInsertReturning queues an INSERT query with RETURNING clause.
// The returned rows will be scanned into the result slice during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var insertedUsers []User
//	insertQuery := psql.Insert(
//	    im.Into("users"),
//	    im.Values(psql.Arg("Alice")),
//	    im.Returning("id", "name", "created_at"),
//	)
//	pgx.QueueInsertReturning(qb, ctx, insertQuery, scan.StructMapper[User](), &insertedUsers)
func QueueInsertReturning[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *[]T) error {
	return QueueSelectAll(qb, ctx, q, m, result)
}

// QueueInsertRowReturning queues an INSERT query with RETURNING that returns exactly one row.
// The returned row will be scanned into the result pointer during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var user User
//	insertQuery := psql.Insert(
//	    im.Into("users"),
//	    im.Values(psql.Arg("Alice")),
//	    im.Returning("id", "name", "created_at"),
//	)
//	pgx.QueueInsertRowReturning(qb, ctx, insertQuery, scan.StructMapper[User](), &user)
func QueueInsertRowReturning[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *T) error {
	return QueueSelectRow(qb, ctx, q, m, result)
}

// QueueUpdateReturning queues an UPDATE query with RETURNING clause.
// The returned rows will be scanned into the result slice during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var updatedUsers []User
//	updateQuery := psql.Update(
//	    um.Table("users"),
//	    um.SetCol("active").ToArg(true),
//	    um.Where(...),
//	    um.Returning("*"),
//	)
//	pgx.QueueUpdateReturning(qb, ctx, updateQuery, scan.StructMapper[User](), &updatedUsers)
func QueueUpdateReturning[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *[]T) error {
	return QueueSelectAll(qb, ctx, q, m, result)
}

// QueueUpdateRowReturning queues an UPDATE query with RETURNING that returns exactly one row.
// The returned row will be scanned into the result pointer during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var user User
//	updateQuery := psql.Update(
//	    um.Table("users"),
//	    um.SetCol("name").ToArg("Bob"),
//	    um.Where(psql.Quote("id").EQ(psql.Arg(1))),
//	    um.Returning("*"),
//	)
//	pgx.QueueUpdateRowReturning(qb, ctx, updateQuery, scan.StructMapper[User](), &user)
func QueueUpdateRowReturning[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *T) error {
	return QueueSelectRow(qb, ctx, q, m, result)
}

// QueueDeleteReturning queues a DELETE query with RETURNING clause.
// The returned rows will be scanned into the result slice during Execute.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var deletedUsers []User
//	deleteQuery := psql.Delete(
//	    dm.From("users"),
//	    dm.Where(psql.Quote("active").EQ(psql.Arg(false))),
//	    dm.Returning("*"),
//	)
//	pgx.QueueDeleteReturning(qb, ctx, deleteQuery, scan.StructMapper[User](), &deletedUsers)
func QueueDeleteReturning[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *[]T) error {
	return QueueSelectAll(qb, ctx, q, m, result)
}

// QueueDeleteRowReturning queues a DELETE query with RETURNING that returns exactly one row.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var user User
//	deleteQuery := psql.Delete(
//	    dm.From("users"),
//	    dm.Where(psql.Quote("id").EQ(psql.Arg(1))),
//	    dm.Returning("*"),
//	)
//	pgx.QueueDeleteRowReturning(qb, ctx, deleteQuery, scan.StructMapper[User](), &user)
func QueueDeleteRowReturning[T any](qb *QueuedBatch, ctx context.Context, q bob.Query, m scan.Mapper[T], result *T) error {
	return QueueSelectRow(qb, ctx, q, m, result)
}

// Len returns the number of queries queued in the batch.
func (qb *QueuedBatch) Len() int {
	return qb.batch.Len()
}

// Execute sends the batch to the database and processes all queued actions.
// All result pointers passed to Queue* helper functions will be populated.
//
// Example:
//
//	qb := pgx.NewQueuedBatch()
//	var user User
//	var count int64
//	pgx.QueueSelectRow(qb, ctx, userQuery, scan.StructMapper[User](), &user)
//	pgx.QueueSelectRow(qb, ctx, countQuery, scan.SingleColumnMapper[int64], &count)
//
//	err := qb.Execute(ctx, tx)
//	if err != nil {
//		return err
//	}
//	// user and count are now populated
func (qb *QueuedBatch) Execute(ctx context.Context, exec Executor) error {
	if qb.batch.Len() == 0 {
		return nil
	}

	br := NewBatchResults(exec.SendBatch(ctx, qb.batch))
	defer br.Close()

	for i, action := range qb.actions {
		if err := action(ctx, br); err != nil {
			return fmt.Errorf("batch action %d failed: %w", i, err)
		}
	}

	return nil
}
