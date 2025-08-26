package pgx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stephenafamo/scan"
)

type transactionBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func beginTx(ctx context.Context, opts pgx.TxOptions, exec transactionBeginner) (Tx, error) {
	ctx, cancel := context.WithCancel(ctx)
	tx, err := exec.BeginTx(ctx, opts)
	if err != nil {
		cancel()
		return Tx{}, err
	}

	// pgx does not automatically rollback the transaction
	// when the context is done, so we do it here
	go func() {
		<-ctx.Done()
		tx.Rollback(ctx)
	}()

	return NewTx(tx, cancel), nil
}

// NewTx wraps an [pgx.Tx] and returns a type that implements [Queryer] but still
// retains the expected methods used by pgx.Tx
// This is useful when an existing pgx.Tx is used in other places in the codebase
// the cancel function is optional, but if provided, it will be called when Commit or Rollback is called
func NewTx(tx pgx.Tx, cancel context.CancelFunc) Tx {
	return Tx{tx, cancel}
}

// Tx is similar to *pgx.Tx but implements [Queryer]
type Tx struct {
	tx     pgx.Tx
	cancel context.CancelFunc
}

// Begin implements pgx.Tx.
func (t Tx) Begin(ctx context.Context) (pgx.Tx, error) {
	ctx, cancel := context.WithCancel(ctx)
	tx, err := t.tx.Begin(ctx)
	if err != nil {
		cancel()
		return Tx{}, err
	}

	// pgx does not automatically rollback the transaction
	// when the context is done, so we do it here
	go func() {
		<-ctx.Done()
		tx.Rollback(ctx)
	}()

	return NewTx(tx, cancel), nil
}

// Conn implements pgx.Tx.
func (t Tx) Conn() *pgx.Conn {
	return t.tx.Conn()
}

// CopyFrom implements pgx.Tx.
func (t Tx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return t.tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// Exec implements pgx.Tx.
func (t Tx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, arguments...)
}

// LargeObjects implements pgx.Tx.
func (t Tx) LargeObjects() pgx.LargeObjects {
	return t.tx.LargeObjects()
}

// Prepare implements pgx.Tx.
func (t Tx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return t.tx.Prepare(ctx, name, sql)
}

// Query implements pgx.Tx.
func (t Tx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

// QueryRow implements pgx.Tx.
func (t Tx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

// SendBatch implements pgx.Tx.
func (t Tx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return t.tx.SendBatch(ctx, b)
}

// Commit implements bob.Transaction.
func (t Tx) Commit(ctx context.Context) error {
	err := t.tx.Commit(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return err
	}

	if t.cancel != nil {
		t.cancel()
	}

	return nil
}

// Rollback implements bob.Transaction.
func (t Tx) Rollback(ctx context.Context) error {
	err := t.tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return err
	}

	if t.cancel != nil {
		t.cancel()
	}

	return nil
}

// ExecContext executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (t Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tag, err := t.tx.Exec(ctx, query, args...)
	return result{tag}, err
}

// QueryContext executes a query that returns rows, typically a SELECT. The args are for any placeholder parameters in the query.
func (t Tx) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	pgxRows, err := t.tx.Query(ctx, query, args...)
	return rows{pgxRows}, err
}
