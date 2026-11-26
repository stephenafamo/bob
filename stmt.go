package bob

import (
	"context"
	"database/sql"
	"slices"

	"github.com/stephenafamo/scan"
)

type Preparer[P PreparedExecutor] interface {
	Executor
	PrepareContext(ctx context.Context, query string) (P, error)
}

type PreparedExecutor interface {
	ExecContext(ctx context.Context, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, args ...any) (scan.Rows, error)
	Close() error
}

// Prepare prepares a query using the [Preparer] and returns a [NamedStmt]
func Prepare[Arg any, P PreparedExecutor](ctx context.Context, exec Preparer[P], q Query) (Stmt[Arg], error) {
	ctx, q, err := resolveForExecution(ctx, exec, q)
	if err != nil {
		return Stmt[Arg]{}, err
	}
	return prepare[Arg](ctx, exec, q)
}

// prepare captures the SQL and loaders after query hooks have run.
func prepare[Arg any, P PreparedExecutor](ctx context.Context, exec Preparer[P], q Query) (Stmt[Arg], error) {
	query, args, err := render(ctx, exec, q, 1)
	if err != nil {
		return Stmt[Arg]{}, err
	}

	binder, err := makeBinder[Arg](args)
	if err != nil {
		return Stmt[Arg]{}, err
	}

	stmt, err := exec.PrepareContext(ctx, string(query))
	if err != nil {
		return Stmt[Arg]{}, err
	}

	s := Stmt[Arg]{
		stmt:    stmt,
		exec:    exec,
		binder:  binder,
		loaders: slices.Clone(loadersFor(ctx, q)),
	}

	return s, nil
}

// Stmt is similar to *sql.Stmt but implements [Queryer]
// instead of taking a list of args, it takes a struct to bind to the query
type Stmt[Arg any] struct {
	stmt    PreparedExecutor
	exec    Executor
	loaders []Loader
	binder  binder[Arg]
}

type txForStmt[Stmt PreparedExecutor] interface {
	Executor
	StmtContext(context.Context, Stmt) Stmt
}

// InTx returns a new MappedStmt that will be executed in the given transaction
func InTx[Arg any, S PreparedExecutor](ctx context.Context, s Stmt[Arg], tx txForStmt[S]) Stmt[Arg] {
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
	ctx = loaderContext(ctx, s.loaders)
	args := s.binder.toArgs(arg)
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

func (s Stmt[Arg]) NamedArgs() []string {
	return s.binder.list()
}

func PrepareQuery[Arg any, P PreparedExecutor, T any](
	ctx context.Context,
	exec Preparer[P],
	q Query,
	m scan.Mapper[T],
) (QueryStmt[Arg, T, []T], error) {
	return PrepareQueryx[Arg, P, T, []T](ctx, exec, q, m)
}

func PrepareQueryx[Arg any, P PreparedExecutor, T any, Ts ~[]T](
	ctx context.Context,
	exec Preparer[P],
	q Query,
	m scan.Mapper[T],
) (QueryStmt[Arg, T, Ts], error) {
	var qs QueryStmt[Arg, T, Ts]

	ctx, q, err := resolveForExecution(ctx, exec, q)
	if err != nil {
		return qs, err
	}
	s, err := prepare[Arg](ctx, exec, q)
	if err != nil {
		return qs, err
	}

	qs = QueryStmt[Arg, T, Ts]{
		Stmt:      s,
		queryType: q.Type(),
		mapper:    queryMapper(ctx, q, m),
	}

	return qs, nil
}

type QueryStmt[Arg, T any, Ts ~[]T] struct {
	Stmt[Arg]

	queryType QueryType
	mapper    scan.Mapper[T]
}

func (s QueryStmt[Arg, T, Ts]) One(ctx context.Context, arg Arg) (T, error) {
	ctx = loaderContext(ctx, s.loaders)
	var t T

	args := s.binder.toArgs(arg)
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

	if h, ok := any(t).(HookableType); ok {
		if err = h.AfterQueryHook(ctx, s.exec, s.queryType); err != nil {
			return t, err
		}
	}

	return t, err
}

func (s QueryStmt[Arg, T, Ts]) All(ctx context.Context, arg Arg) (Ts, error) {
	ctx = loaderContext(ctx, s.loaders)
	// Mirror Allx: if the slice type is not hookable but the single type
	// is, reject before executing the query (see ErrHookableTypeMismatch).
	_, isSliceHookable := any(*new(Ts)).(HookableType)
	_, isSingleHookable := any(*new(T)).(HookableType)
	if !isSliceHookable && isSingleHookable {
		return nil, ErrHookableTypeMismatch
	}

	args := s.binder.toArgs(arg)
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

	if h, ok := any(typedSlice).(HookableType); ok {
		if err = h.AfterQueryHook(ctx, s.exec, s.queryType); err != nil {
			return typedSlice, err
		}
	}

	return typedSlice, err
}

func (s QueryStmt[Arg, T, Ts]) Cursor(ctx context.Context, arg Arg) (scan.ICursor[T], error) {
	ctx = loaderContext(ctx, s.loaders)
	args := s.binder.toArgs(arg)
	rows, err := s.stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}

	_, isHookable := any(*new(T)).(HookableType)

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

			if isHookable {
				if err = any(t).(HookableType).AfterQueryHook(ctx, s.exec, s.queryType); err != nil {
					return t, err
				}
			}

			return t, err
		}
	})

	return scan.CursorFromRows(ctx, m2, rows)
}
