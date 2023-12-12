package bob

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
)

// NewQueryer wraps an [stdscan.Queryer] and makes it a [scan.Queryer]
func NewQueryer[T stdscan.Queryer](wrapped T) scan.Queryer {
	return commonQueryer[T]{wrapped: wrapped}
}

type commonQueryer[T stdscan.Queryer] struct {
	wrapped T
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (q commonQueryer[T]) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return q.wrapped.QueryContext(ctx, query, args...)
}

// StdInterface is an interface that *sql.DB, *sql.Tx and *sql.Conn satisfy
type StdInterface interface {
	stdscan.Queryer
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// New wraps an stdInterface to make it comply with Queryer
// It also includes a number of other methods that are often used with
// *sql.DB, *sql.Tx and *sql.Conn
func New[T StdInterface](wrapped T) common[T] {
	return common[T]{commonQueryer[T]{wrapped: wrapped}}
}

type common[T StdInterface] struct {
	commonQueryer[T]
}

// PrepareContext creates a prepared statement for later queries or executions
func (c common[T]) PrepareContext(ctx context.Context, query string) (StdPrepared, error) {
	s, err := c.wrapped.PrepareContext(ctx, query)
	return StdPrepared{s}, err
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (q common[T]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.wrapped.ExecContext(ctx, query, args...)
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

// PingContext verifies a connection to the database is still alive, establishing a connection if necessary.
func (d DB) PingContext(ctx context.Context) error {
	return d.wrapped.PingContext(ctx)
}

// Close works the same as [*sql.DB.Close]
func (d DB) Close() error {
	return d.wrapped.Close()
}

// BeginTx is similar to [*sql.DB.BeginTx], but return a transaction that
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

var (
	_ txForStmt[StdPrepared] = &Tx{}
	_ Preparer[StdPrepared]  = &Tx{}
)

// Tx is similar to *sql.Tx but implements [Queryer]
type Tx struct {
	common[*sql.Tx]
}

// Commit works the same as [*sql.Tx.Commit]
func (t Tx) Commit() error {
	return t.wrapped.Commit()
}

// Rollback works the same as [*sql.Tx.Rollback]
func (t Tx) Rollback() error {
	return t.wrapped.Rollback()
}

func (tx *Tx) StmtContext(ctx context.Context, stmt StdPrepared) StdPrepared {
	return StdPrepared{tx.wrapped.StmtContext(ctx, stmt.Stmt)}
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

// PingContext verifies a connection to the database is still alive, establishing a connection if necessary.
func (c Conn) PingContext(ctx context.Context) error {
	return c.wrapped.PingContext(ctx)
}

// Close works the same as [*sql.Conn.Close]
func (c Conn) Close() error {
	return c.wrapped.Close()
}

// BeginTx is similar to [*sql.Conn.BeginTx], but return a transaction that
// implements [Queryer]
func (c Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := c.wrapped.BeginTx(ctx, opts)
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
