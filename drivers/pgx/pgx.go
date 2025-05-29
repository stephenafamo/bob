package pgx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

// New works just like [pgxpool.New], but converts the returned [*pgxpool.Pool] to [Pool]
func New(ctx context.Context, dsn string) (Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	return NewPool(pool), err
}

// NewWithConfig works just like [pgxpool.NewWithConfig], but converts the returned [*pgxpool.Pool] to [Pool]
func NewWithConfig(ctx context.Context, config *pgxpool.Config) (Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, config)
	return NewPool(pool), err
}

// NewPool wraps an [*pgxpool.Pool] and returns a type that implements [bob.Executor] but still
// retains the expected methods used by *pgxpool.Pool
// This is useful when an existing *pgxpool.Pool is used in other places in the codebase
func NewPool(pool *pgxpool.Pool) Pool {
	return Pool{pool}
}

// Pool is similar to *pgxpool.Pool but implement [Queryer]
type Pool struct {
	*pgxpool.Pool
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (p Pool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tag, err := p.Pool.Exec(ctx, query, args...)
	return result{tag}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (p Pool) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	pgxRows, err := p.Pool.Query(ctx, query, args...)
	return rows{pgxRows}, err
}

// Begin is similar to [*pgxpool.Pool.Begin], but return a transaction that
// implements [Queryer]
func (p Pool) Begin(ctx context.Context) (bob.Transaction, error) {
	return p.BeginTx(ctx, pgx.TxOptions{})
}

// BeginTx is similar to [*pgxpool.Pool.BeginTx], but return a transaction that
// implements [Queryer]
func (p Pool) BeginTx(ctx context.Context, opts pgx.TxOptions) (bob.Transaction, error) {
	tx, err := p.Pool.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}

	return NewTx(tx), nil
}

// NewTx wraps an [*pgx.Tx] and returns a type that implements [Queryer] but still
// retains the expected methods used by *pgx.Tx
// This is useful when an existing *pgx.Tx is used in other places in the codebase
func NewTx(tx pgx.Tx) Tx {
	return Tx{tx}
}

// Tx is similar to *pgx.Tx but implements [Queryer]
type Tx struct {
	pgx.Tx
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (t Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tag, err := t.Tx.Exec(ctx, query, args...)
	return result{tag}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (t Tx) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	pgxRows, err := t.Tx.Query(ctx, query, args...)
	return rows{pgxRows}, err
}

// NewConn wraps an [*pgx.Conn] and returns a type that implements [Queryer]
// This is useful when an existing *pgx.Conn is used in other places in the codebase
func NewConn(conn *pgx.Conn) Conn {
	return Conn{conn}
}

// Conn is similar to *pgx.Conn but implements [Queryer]
type Conn struct {
	*pgx.Conn
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (c Conn) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tag, err := c.Conn.Exec(ctx, query, args...)
	return result{tag}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (c Conn) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	pgxRows, err := c.Conn.Query(ctx, query, args...)
	return rows{pgxRows}, err
}

// Begin is similar to [*pgxpool.Pool.Begin], but return a transaction that
// implements [Queryer]
func (c Conn) Begin(ctx context.Context) (bob.Transaction, error) {
	return c.BeginTx(ctx, pgx.TxOptions{})
}

// BeginTx is similar to [*pgxpool.Pool.BeginTx], but return a transaction that
// implements [Queryer]
func (c Conn) BeginTx(ctx context.Context, opts pgx.TxOptions) (bob.Transaction, error) {
	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}

	return NewTx(tx), nil
}

type result struct {
	pgconn.CommandTag
}

// LastInsertId implements sql.Result.
func (r result) LastInsertId() (int64, error) {
	return 0, errors.New("pgx does not support LastInsertId")
}

// RowsAffected implements sql.Result.
func (r result) RowsAffected() (int64, error) {
	return r.CommandTag.RowsAffected(), nil
}

type rows struct {
	pgx.Rows
}

func (r rows) Close() error {
	r.Rows.Close()
	return nil
}

func (r rows) Columns() ([]string, error) {
	fields := r.FieldDescriptions()
	cols := make([]string, len(fields))

	for i, field := range fields {
		cols[i] = field.Name
	}

	return cols, nil
}
