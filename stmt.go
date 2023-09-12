package bob

import (
	"context"
	"database/sql"

	"github.com/stephenafamo/scan"
)

type Preparer[P Prepared] interface {
	Executor
	PrepareContext(ctx context.Context, query string) (P, error)
}

type Prepared interface {
	ExecContext(ctx context.Context, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, args ...any) (scan.Rows, error)
	Close() error
}

// Prepare prepares a query using the [Preparer] and returns a [NamedStmt]
func Prepare[Arg any, P Prepared](ctx context.Context, exec Preparer[P], q Query) (Stmt[Arg], error) {
	bq, err := BindNamed[Arg](q)
	if err != nil {
		return Stmt[Arg]{}, err
	}

	stmt, err := exec.PrepareContext(ctx, string(bq.query))
	if err != nil {
		return Stmt[Arg]{}, err
	}

	s := Stmt[Arg]{
		stmt:   stmt,
		exec:   exec,
		binder: bq.binder,
	}

	if l, ok := q.(Loadable); ok {
		loaders := l.GetLoaders()
		s.loaders = make([]Loader, len(loaders))
		copy(s.loaders, loaders)
	}

	return s, nil
}

// Stmt is similar to *sql.Stmt but implements [Queryer]
// instead of taking a list of args, it takes a struct to bind to the query
type Stmt[Arg any] struct {
	stmt    Prepared
	exec    Executor
	loaders []Loader
	binder  binder[Arg]
}

type txForStmt[Stmt Prepared] interface {
	Executor
	StmtContext(context.Context, Stmt) Stmt
}

// InTx returns a new MappedStmt that will be executed in the given transaction
func InTx[Arg any, S Prepared](ctx context.Context, s Stmt[Arg], tx txForStmt[S]) Stmt[Arg] {
	stmt, ok := s.stmt.(S)
	if !ok {
		panic("stmt is not an the right type")
	}

	s.stmt = tx.StmtContext(ctx, stmt)
	s.exec = tx
	return s
}

// Close closes the statement.
func (s Stmt[Arg]) Close() error {
	return s.stmt.Close()
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s Stmt[Arg]) Exec(ctx context.Context, arg Arg) (sql.Result, error) {
	args := s.binder.ToArgs(arg)
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

func PrepareQuery[Arg any, P Prepared, T any](ctx context.Context, exec Preparer[P], q Query, m scan.Mapper[T], opts ...ExecOption[T]) (QueryStmt[Arg, T, []T], error) {
	return PrepareQueryx[Arg, P, T, []T](ctx, exec, q, m, opts...)
}

func PrepareQueryx[Arg any, P Prepared, T any, Ts ~[]T](ctx context.Context, exec Preparer[P], q Query, m scan.Mapper[T], opts ...ExecOption[T]) (QueryStmt[Arg, T, Ts], error) {
	var qs QueryStmt[Arg, T, Ts]

	s, err := Prepare[Arg, P](ctx, exec, q)
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

	qs = QueryStmt[Arg, T, Ts]{
		Stmt:     s,
		mapper:   m,
		settings: settings,
	}

	return qs, nil
}

type QueryStmt[Arg, T any, Ts ~[]T] struct {
	Stmt[Arg]

	mapper   scan.Mapper[T]
	settings ExecSettings[T]
}

func (s QueryStmt[Arg, T, Ts]) One(ctx context.Context, arg Arg) (T, error) {
	var t T

	args := s.binder.ToArgs(arg)
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

func (s QueryStmt[Arg, T, Ts]) All(ctx context.Context, arg Arg) (Ts, error) {
	args := s.binder.ToArgs(arg)
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

func (s QueryStmt[Arg, T, Ts]) Cursor(ctx context.Context, arg Arg) (scan.ICursor[T], error) {
	args := s.binder.ToArgs(arg)
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
