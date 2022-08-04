package bob

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

var (
	_ db = DB{}
	_ db = Tx{}
)

// A constraint that *sql.DB, *sql.Tx and *sql.Conn satisfy
type stdInterface interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func New[T stdInterface](wrapped T) Common[T] {
	return Common[T]{wrapped: wrapped}
}

type Common[T stdInterface] struct {
	wrapped T
}

func (q Common[T]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.wrapped.ExecContext(ctx, query, args...)
}

func (q Common[T]) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return q.wrapped.QueryContext(ctx, query, args...)
}

func (q Common[T]) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return q.wrapped.QueryRowContext(ctx, query, args...)
}

func Open(driverName string, dataSource string) (DB, error) {
	db, err := sql.Open(driverName, dataSource)
	return NewDB(db), err
}

func OpenDB(c driver.Connector) DB {
	return NewDB(sql.OpenDB(c))
}

func NewDB(db *sql.DB) DB {
	return DB{Common[*sql.DB]{wrapped: db}}
}

type DB struct {
	Common[*sql.DB]
}

func (d DB) Close() error {
	return d.wrapped.Close()
}

func (d DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := d.wrapped.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}

	return NewTx(tx), nil
}

func NewTx(tx *sql.Tx) Tx {
	return Tx{
		Common: Common[*sql.Tx]{wrapped: tx},
	}
}

type Tx struct {
	Common[*sql.Tx]
}

func (t Tx) Commit() error {
	return t.wrapped.Commit()
}

func (t Tx) Rollback() error {
	return t.wrapped.Rollback()
}
