package bob

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/stephenafamo/scan"
)

type NamedArgRequiredError struct{ value any }

func (e NamedArgRequiredError) Error() string {
	return fmt.Sprintf("expected named arg, got %#v", e.value)
}

type DuplicateArgError struct{ Name string }

func (e DuplicateArgError) Error() string {
	return fmt.Sprintf("duplicate arg %s", e.Name)
}

type MissingArgError struct{ Name string }

func (e MissingArgError) Error() string {
	return fmt.Sprintf("missing arg %s", e.Name)
}

type mapBinder struct {
	unique    int
	positions []string
}

func (m mapBinder) toArgs(mapArgs map[string]any) ([]any, error) {
	if len(mapArgs) != m.unique {
		return nil, MismatchedArgsError{
			Expected: m.unique,
			Got:      len(mapArgs),
		}
	}

	args := make([]any, len(m.positions))
	for position, name := range m.positions {
		value, ok := mapArgs[name]
		if !ok {
			return nil, MissingArgError{Name: name}
		}

		args[position] = value
	}

	return args, nil
}

func makeMapBinder(args []any) (mapBinder, error) {
	positions := make([]string, len(args))
	for pos, arg := range args {
		if name, ok := arg.(namedArg); ok {
			positions[pos] = string(name)
			continue
		}

		if name, ok := arg.(named); ok && len(name.names) == 1 {
			positions[pos] = string(name.names[0])
			continue
		}

		return mapBinder{}, NamedArgRequiredError{arg}
	}

	// count unique names
	unique := make(map[string]struct{})
	for _, name := range positions {
		if _, ok := unique[name]; !ok {
			unique[name] = struct{}{}
		}
	}

	return mapBinder{
		unique:    len(unique),
		positions: positions,
	}, nil
}

// PrepareMapped prepares a query using the [Preparer] and returns a [NamedStmt]
func PrepareMapped(ctx context.Context, exec Preparer, q Query) (MappedStmt, error) {
	stmt, args, err := prepare(ctx, exec, q)
	if err != nil {
		return MappedStmt{}, err
	}

	m, err := makeMapBinder(args)
	if err != nil {
		return MappedStmt{}, err
	}

	return MappedStmt{
		stmt:   stmt,
		mapper: m,
	}, nil
}

// MappedStmt is similar to *sql.Stmt but implements [Queryer]
// instead of taking a list of args, it takes a map of args or a struct to bind to the query
type MappedStmt struct {
	stmt   Stmt
	mapper mapBinder
}

// Inspect returns a map with all the expected keys
func (s MappedStmt) Inspect() []string {
	return s.mapper.positions
}

// Close closes the statement
func (s MappedStmt) Close() error {
	return s.stmt.Close()
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s MappedStmt) Exec(ctx context.Context, mappedArgs map[string]any) (sql.Result, error) {
	args, err := s.mapper.toArgs(mappedArgs)
	if err != nil {
		return nil, err
	}

	return s.stmt.Exec(ctx, args...)
}

func PrepareMappedQuery[T any](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (MappedQueryStmt[T, []T], error) {
	return PrepareMappedQueryx[T, []T](ctx, exec, q, m, opts...)
}

func PrepareMappedQueryx[T any, Ts ~[]T](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (MappedQueryStmt[T, Ts], error) {
	s, args, err := prepareQuery[T, Ts](ctx, exec, q, m, opts...)
	if err != nil {
		return MappedQueryStmt[T, Ts]{}, err
	}

	binder, err := makeMapBinder(args)
	if err != nil {
		return MappedQueryStmt[T, Ts]{}, err
	}

	return MappedQueryStmt[T, Ts]{
		query:  s,
		binder: binder,
	}, nil
}

type MappedQueryStmt[T any, Ts ~[]T] struct {
	query  QueryStmt[T, Ts]
	binder mapBinder
}

// Close closes the statement
func (s MappedQueryStmt[T, Ts]) Close() error {
	return s.query.Close()
}

func (s MappedQueryStmt[T, Ts]) One(ctx context.Context, arg map[string]any) (T, error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		var t T
		return t, err
	}

	return s.query.One(ctx, args...)
}

func (s MappedQueryStmt[T, Ts]) All(ctx context.Context, arg map[string]any) (Ts, error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		return nil, err
	}

	return s.query.All(ctx, args...)
}

func (s MappedQueryStmt[T, Ts]) Cursor(ctx context.Context, arg map[string]any) (scan.ICursor[T], error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		return nil, err
	}

	return s.query.Cursor(ctx, args...)
}
