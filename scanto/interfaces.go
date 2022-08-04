package scanto

import (
	"context"
	"database/sql"
)

type db interface {
	Execer
	Queryer
}

type Queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Row interface {
	Scan(...any) error
}

type Rows interface {
	Row
	Columns() ([]string, error)
	Next() bool
	Close() error
	Err() error
}
