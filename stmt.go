package bob

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/stephenafamo/scan"
)

type MismatchedArgsError struct {
	Expected int
	Got      int
}

func (e MismatchedArgsError) Error() string {
	return fmt.Sprintf("expected %d args, got %d", e.Expected, e.Got)
}

type Preparer interface {
	Executor
	PrepareContext(ctx context.Context, query string) (Statement, error)
}

type Statement interface {
	ExecContext(ctx context.Context, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, args ...any) (scan.Rows, error)
	Close() error
}

func prepare(ctx context.Context, exec Preparer, q Query) (Stmt, []any, error) {
	query, args, err := Build(q)
	if err != nil {
		return Stmt{}, nil, err
	}

	stmt, err := exec.PrepareContext(ctx, query)
	if err != nil {
		return Stmt{}, nil, err
	}

	s := Stmt{
		exec: exec,
		stmt: stmt,
	}

	if l, ok := q.(Loadable); ok {
		loaders := l.GetLoaders()
		s.loaders = make([]Loader, len(loaders))
		copy(s.loaders, loaders)
	}

	return s, args, nil
}

// Prepare prepares a query using the [Preparer] and returns a [Stmt]
func Prepare(ctx context.Context, exec Preparer, q Query) (Stmt, error) {
	s, _, err := prepare(ctx, exec, q)
	return s, err
}

// Stmt is similar to *sql.Stmt but implements [Queryer]
type Stmt struct {
	stmt    Statement
	exec    Executor
	loaders []Loader
}

// Close closes the statement
func (s Stmt) Close() error {
	return s.stmt.Close()
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s Stmt) Exec(ctx context.Context, args ...any) (sql.Result, error) {
	result, err := s.stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, err
	}

	for _, loader := range s.loaders {
		if err := loader.Load(ctx, s.exec, nil); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func PrepareQuery[T any](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (QueryStmt[T, []T], error) {
	return PrepareQueryx[T, []T](ctx, exec, q, m, opts...)
}

func PrepareQueryx[T any, Ts ~[]T](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (QueryStmt[T, Ts], error) {
	s, _, err := prepareQuery[T, Ts](ctx, exec, q, m, opts...)
	return s, err
}

func prepareQuery[T any, Ts ~[]T](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (QueryStmt[T, Ts], []any, error) {
	var qs QueryStmt[T, Ts]

	s, args, err := prepare(ctx, exec, q)
	if err != nil {
		return qs, nil, err
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

	return qs, args, nil
}

type QueryStmt[T any, Ts ~[]T] struct {
	Stmt

	mapper   scan.Mapper[T]
	settings ExecSettings[T]
}

// Close closes the statement
func (s QueryStmt[T, Ts]) Close() error {
	return s.stmt.Close()
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

func (s QueryStmt[T, Ts]) Cursor(ctx context.Context, args ...any) (scan.ICursor[T], error) {
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}

	m2 := scan.Mapper[T](func(ctx context.Context, c []string) (scan.BeforeFunc, func(any) (T, error)) {
		before, after := s.mapper(ctx, c)
		return before, func(link any) (T, error) {
			t, err := after(link)
			if err != nil {
				return t, err
			}

			for _, loader := range s.loaders {
				err = loader.Load(ctx, s.exec, t)
				if err != nil {
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
	})

	return scan.CursorFromRows(ctx, m2, rows)
}
