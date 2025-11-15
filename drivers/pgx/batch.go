package pgx

import (
	"context"
	"database/sql"
	"errors"

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
