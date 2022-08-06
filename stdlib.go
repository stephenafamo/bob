package bob

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/stephenafamo/scan"
)

// A Queryer that returns the concrete type *sql.Rows
type StdQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// NewQueryer wraps an StdQueryer and makes it a Queryer
func NewQueryer[T StdQueryer](wrapped T) scan.Queryer {
	return commonQueryer[T]{wrapped: wrapped}
}

type commonQueryer[T StdQueryer] struct {
	wrapped T
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (q commonQueryer[T]) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return q.wrapped.QueryContext(ctx, query, args...)
}

// An interface that *sql.DB, *sql.Tx and *sql.Conn satisfy
type StdInterface interface {
	StdQueryer
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// New wraps an StdInterface to make it comply with Queryer
// It also includes a number of other methods that are often used with
// *sql.DB, *sql.Tx and *sql.Conn
func New[T StdInterface](wrapped T) common[T] {
	return common[T]{commonQueryer[T]{wrapped: wrapped}}
}

type common[T StdInterface] struct {
	commonQueryer[T]
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (q common[T]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.wrapped.ExecContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row. QueryRowContext always returns a non-nil value. Errors are deferred until Row's Scan method is called. If the query selects no rows, the *Row's Scan will return ErrNoRows. Otherwise, the *Row's Scan scans the first selected row and discards the rest.
func (q common[T]) QueryRowContext(ctx context.Context, query string, args ...any) scan.Row {
	return q.wrapped.QueryRowContext(ctx, query, args...)
}

// PrepareContext creates a prepared statement for later queries or executions. Multiple queries or executions may be run concurrently from the returned statement. The caller must call the statement's Close method when the statement is no longer needed.
//
// The provided context is used for the preparation of the statement, not for the execution of the statement.
func (q common[T]) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return q.wrapped.PrepareContext(ctx, query)
}

// Open works just like [sql.Open], but converts the returned [*sql.DB] to [DB]
func Open(driverName string, dataSource string) (DB, error) {
	db, err := sql.Open(driverName, dataSource)
	return NewDB(db), err
}

// OpenDB works just like [sql.OpenDB], but converts the returned [*sql.DB] to [DB]
func OpenDB(c driver.Connector) DB {
	return NewDB(sql.OpenDB(c))
}

// NewDB wraps an [*sql.DB] and returns a type that implements [Queryer] but still
// retains the expected methods used by *sql.DB
// This is useful when an existing *sql.DB is used in other places in the codebase
func NewDB(db *sql.DB) DB {
	return DB{common: New(db)}
}

// DB is similar to *sql.DB but implement [Queryer]
type DB struct {
	common[*sql.DB]
}

// Close works the same as [sql.DB.Close()]
func (d DB) Close() error {
	return d.wrapped.Close()
}

// BeginTx is similar to [sql.DB.BeginTx()], but return a transaction that
// implements [Queryer]
func (d DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := d.wrapped.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}

	return NewTx(tx), nil
}

// NewTx wraps an [*sql.Tx] and returns a type that implements [Queryer] but still
// retains the expected methods used by *sql.Tx
// This is useful when an existing *sql.Tx is used in other places in the codebase
func NewTx(tx *sql.Tx) Tx {
	return Tx{New(tx)}
}

// Tx is similar to *sql.Tx but implements [Queryer]
type Tx struct {
	common[*sql.Tx]
}

// Commit works the same as [*sql.Tx.Commit()]
func (t Tx) Commit() error {
	return t.wrapped.Commit()
}

// Rollback works the same as [*sql.Tx.Rollback()]
func (t Tx) Rollback() error {
	return t.wrapped.Rollback()
}

// NewConn wraps an [*sql.Conn] and returns a type that implements [Queryer]
// This is useful when an existing *sql.Conn is used in other places in the codebase
func NewConn(conn *sql.Conn) Conn {
	return Conn{New(conn)}
}

// Conn is similar to *sql.Conn but implements [Queryer]
type Conn struct {
	common[*sql.Conn]
}
