package bob

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/stephenafamo/scan"
)

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
	return DB{db}
}

// DB is similar to *sql.DB but implement [Queryer]
type DB struct {
	*sql.DB
}

// PrepareContext creates a prepared statement for later queries or executions
func (d DB) PrepareContext(ctx context.Context, query string) (StdPrepared, error) {
	s, err := d.DB.PrepareContext(ctx, query)
	return StdPrepared{s}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (d DB) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}

// Begin is similar to [*sql.DB.BeginTx], but return a transaction that
// implements [Queryer]
func (d DB) Begin(ctx context.Context) (Transaction, error) {
	return d.BeginTx(ctx, nil)
}

// BeginTx is similar to [*sql.DB.BeginTx], but return a transaction that
// implements [Queryer]
func (d DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}

	return NewTx(tx), nil
}

// RunInTx runs the provided function in a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (d DB) RunInTx(ctx context.Context, txOptions *sql.TxOptions, fn func(context.Context, Executor) error) error {
	tx, err := d.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		err = fmt.Errorf("call: %w", err)

		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// NewTx wraps an [*sql.Tx] and returns a type that implements [Queryer] but still
// retains the expected methods used by *sql.Tx
// This is useful when an existing *sql.Tx is used in other places in the codebase
func NewTx(tx *sql.Tx) Tx {
	return Tx{tx}
}

// Tx is similar to *sql.Tx but implements [Queryer]
type Tx struct {
	*sql.Tx
}

// PrepareContext creates a prepared statement for later queries or executions
func (t Tx) PrepareContext(ctx context.Context, query string) (StdPrepared, error) {
	s, err := t.Tx.PrepareContext(ctx, query)
	return StdPrepared{s}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (t Tx) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return t.Tx.QueryContext(ctx, query, args...)
}

// Commit works the same as [*sql.Tx.Commit]
func (t Tx) Commit(_ context.Context) error {
	return t.Tx.Commit()
}

// Rollback works the same as [*sql.Tx.Rollback]
func (t Tx) Rollback(_ context.Context) error {
	return t.Tx.Rollback()
}

func (tx Tx) StmtContext(ctx context.Context, stmt StdPrepared) StdPrepared {
	return StdPrepared{tx.Tx.StmtContext(ctx, stmt.Stmt)}
}

// NewConn wraps an [*sql.Conn] and returns a type that implements [Queryer]
// This is useful when an existing *sql.Conn is used in other places in the codebase
func NewConn(conn *sql.Conn) Conn {
	return Conn{conn}
}

// Conn is similar to *sql.Conn but implements [Queryer]
type Conn struct {
	*sql.Conn
}

// PrepareContext creates a prepared statement for later queries or executions
func (d Conn) PrepareContext(ctx context.Context, query string) (StdPrepared, error) {
	s, err := d.Conn.PrepareContext(ctx, query)
	return StdPrepared{s}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (d Conn) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return d.Conn.QueryContext(ctx, query, args...)
}

// Begin is similar to [*sql.DB.BeginTx], but return a transaction that
// implements [Queryer]
func (d Conn) Begin(ctx context.Context) (Transaction, error) {
	return d.BeginTx(ctx, nil)
}

// BeginTx is similar to [*sql.DB.BeginTx], but return a transaction that
// implements [Queryer]
func (d Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := d.Conn.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}

	return NewTx(tx), nil
}

type StdPrepared struct {
	*sql.Stmt
}

func (s StdPrepared) QueryContext(ctx context.Context, args ...any) (scan.Rows, error) {
	return s.Stmt.QueryContext(ctx, args...)
}
