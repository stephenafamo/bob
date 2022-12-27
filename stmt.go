package bob

import (
	"context"
	"database/sql"

	"github.com/stephenafamo/scan"
)

type Preparer interface {
	Executor
	PrepareContext(ctx context.Context, query string) (Statement, error)
}

type Statement interface {
	ExecContext(ctx context.Context, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, args ...any) (scan.Rows, error)
}

// NewStmt wraps an [*sql.Stmt] and returns a type that implements [Queryer] but still
// retains the expected methods used by *sql.Stmt
// This is useful when an existing *sql.Stmt is used in other places in the codebase
func Prepare(ctx context.Context, q Query, exec Preparer) (Stmt, error) {
	query, args, err := Build(q)
	if err != nil {
		return Stmt{}, err
	}

	stmt, err := exec.PrepareContext(ctx, query)
	if err != nil {
		return Stmt{}, err
	}

	s := Stmt{
		exec:    exec,
		stmt:    stmt,
		lenArgs: len(args),
	}

	if l, ok := q.(Loadable); ok {
		loaders := l.GetLoaders()
		s.loaders = make([]Loader, len(loaders))
		copy(s.loaders, loaders)
	}

	return s, nil
}

// Stmt is similar to *sql.Stmt but implements [Queryer]
type Stmt struct {
	stmt    Statement
	exec    Executor
	lenArgs int
	loaders []Loader
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s Stmt) Exec(ctx context.Context, args ...any) (int64, error) {
	result, err := s.stmt.ExecContext(ctx, args...)
	if err != nil {
		return 0, err
	}

	for _, loader := range s.loaders {
		if err := loader.Load(ctx, s.exec, nil); err != nil {
			return 0, err
		}
	}

	return result.RowsAffected()
}

func PrepareQuery[T any](ctx context.Context, q Query, m scan.Mapper[T], exec Preparer, opts ...ExecOption[T]) (QueryStmt[T, []T], error) {
	return PrepareQueryx[T, []T](ctx, q, m, exec, opts...)
}

func PrepareQueryx[T any, Ts ~[]T](ctx context.Context, q Query, m scan.Mapper[T], exec Preparer, opts ...ExecOption[T]) (QueryStmt[T, Ts], error) {
	var qs QueryStmt[T, Ts]

	s, err := Prepare(ctx, q, exec)
	if err != nil {
		return qs, err
	}

	settings := ExecSettings[T]{}
	for _, opt := range opts {
		opt(&settings)
	}

	if l, ok := q.(MapperModder); ok {
		if loaders := l.GetMapperMods(); len(loaders) > 0 {
			m = scan.Mod(m, loaders...)
		}
	}

	qs = QueryStmt[T, Ts]{
		Stmt:     s,
		mapper:   m,
		settings: settings,
	}

	return qs, nil
}

type QueryStmt[T any, Ts ~[]T] struct {
	Stmt

	mapper   scan.Mapper[T]
	settings ExecSettings[T]
}

func (s QueryStmt[T, Ts]) One(ctx context.Context, args ...any) (T, error) {
	var t T

	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return t, err
	}

	t, err = scan.OneFromRows(ctx, s.mapper, rows)
	if err != nil {
		return t, err
	}

	for _, loader := range s.loaders {
		if err := loader.Load(ctx, s.exec, t); err != nil {
			return t, err
		}
	}

	if s.settings.AfterSelect != nil {
		if err := s.settings.AfterSelect(ctx, []T{t}); err != nil {
			return t, err
		}
	}

	return t, err
}

func (s QueryStmt[T, Ts]) All(ctx context.Context, args ...any) (Ts, error) {
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}

	rawSlice, err := scan.AllFromRows(ctx, s.mapper, rows)
	if err != nil {
		return nil, err
	}

	typedSlice := Ts(rawSlice)

	for _, loader := range s.loaders {
		if err := loader.Load(ctx, s.exec, typedSlice); err != nil {
			return nil, err
		}
	}

	if s.settings.AfterSelect != nil {
		if err := s.settings.AfterSelect(ctx, typedSlice); err != nil {
			return nil, err
		}
	}

	return typedSlice, err
}
