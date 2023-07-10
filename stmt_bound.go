package bob

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/scan"
)

type structBinder[T any] struct {
	args   []string
	fields []string
}

func (b structBinder[T]) toArgs(arg T) ([]any, error) {
	val := reflect.ValueOf(arg)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, errors.New("object is nil")
		}
		val = val.Elem()
	}

	values := make([]any, len(b.args))

ArgLoop:
	for index, argName := range b.args {
		for _, fieldName := range b.fields {
			if fieldName == argName {
				field := val.Field(index)
				values[index] = field.Interface()
				continue ArgLoop
			}
		}
		return nil, ErrMissingArg{Name: argName}
	}

	return values, nil
}

func makeStructBinder[Arg any](args []any) (structBinder[Arg], error) {
	argPositions := make([]string, len(args))
	for pos, arg := range args {
		if name, ok := arg.(namedArg); ok {
			argPositions[pos] = string(name)
			continue
		}

		if name, ok := arg.(named); ok && len(name.names) == 1 {
			argPositions[pos] = string(name.names[0])
			continue
		}

		return structBinder[Arg]{}, ErrNamedArgRequired{arg}
	}

	var x Arg
	fieldPositions := mappings.GetMappings(reflect.TypeOf(x)).All

	// check if all positions have matching fields
ArgLoop:
	for _, name := range argPositions {
		for _, field := range fieldPositions {
			if field == name {
				continue ArgLoop
			}
		}
		return structBinder[Arg]{}, ErrMissingArg{Name: name}
	}

	return structBinder[Arg]{
		args:   argPositions,
		fields: fieldPositions,
	}, nil
}

// PrepareBound prepares a query using the [Preparer] and returns a [NamedStmt]
func PrepareBound[Arg any](ctx context.Context, exec Preparer, q Query) (BoundStmt[Arg], error) {
	stmt, args, err := prepare(ctx, exec, q)
	if err != nil {
		return BoundStmt[Arg]{}, err
	}

	binder, err := makeStructBinder[Arg](args)
	if err != nil {
		return BoundStmt[Arg]{}, err
	}

	return BoundStmt[Arg]{
		stmt:   stmt,
		binder: binder,
	}, nil
}

// BoundStmt is similar to *sql.Stmt but implements [Queryer]
// instead of taking a list of args, it takes a struct to bind to the query
type BoundStmt[Arg any] struct {
	stmt   Stmt
	binder structBinder[Arg]
}

// Close closes the statement.
func (s BoundStmt[Arg]) Close() error {
	return s.stmt.Close()
}

// Exec executes a query without returning any rows. The args are for any placeholder parameters in the query.
func (s BoundStmt[Arg]) Exec(ctx context.Context, arg Arg) (sql.Result, error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		return nil, err
	}

	return s.stmt.Exec(ctx, args...)
}

func PrepareBoundQuery[Arg any, T any](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (BoundQueryStmt[Arg, T, []T], error) {
	return PrepareBoundQueryx[Arg, T, []T](ctx, exec, q, m, opts...)
}

func PrepareBoundQueryx[Arg any, T any, Ts ~[]T](ctx context.Context, exec Preparer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (BoundQueryStmt[Arg, T, Ts], error) {
	s, args, err := prepareQuery[T, Ts](ctx, exec, q, m, opts...)

	binder, err := makeStructBinder[Arg](args)
	if err != nil {
		return BoundQueryStmt[Arg, T, Ts]{}, err
	}

	return BoundQueryStmt[Arg, T, Ts]{
		query:  s,
		binder: binder,
	}, nil
}

type BoundQueryStmt[Arg any, T any, Ts ~[]T] struct {
	query  QueryStmt[T, Ts]
	binder structBinder[Arg]
}

// Close closes the statment.
func (s BoundQueryStmt[Arg, T, Ts]) Close() error {
	return s.query.Close()
}

func (s BoundQueryStmt[Arg, T, Ts]) One(ctx context.Context, arg Arg) (T, error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		var t T
		return t, err
	}

	return s.query.One(ctx, args...)
}

func (s BoundQueryStmt[Arg, T, Ts]) All(ctx context.Context, arg Arg) (Ts, error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		return nil, err
	}

	return s.query.All(ctx, args...)
}

func (s BoundQueryStmt[Arg, T, Ts]) Cursor(ctx context.Context, arg Arg) (scan.ICursor[T], error) {
	args, err := s.binder.toArgs(arg)
	if err != nil {
		return nil, err
	}

	return s.query.Cursor(ctx, args...)
}
