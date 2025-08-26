package bob

import (
	"context"
	"database/sql"
	"errors"

	"github.com/stephenafamo/scan"
)

var ErrHookableTypeMismatch = errors.New("hookable type mismatch: the slice type is not hookable, but the single type is")

type (
	MapperModder interface {
		GetMapperMods() []scan.MapperMod
	}

	HookableQuery interface {
		RunHooks(context.Context, Executor) (context.Context, error)
	}

	// If a type implements this interface,
	// it will be called after the query has been executed and it is scanned
	HookableType interface {
		AfterQueryHook(context.Context, Executor, QueryType) error
	}
)

type Executor interface {
	scan.Queryer
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Transactor[Tx Transaction] interface {
	Executor
	Begin(context.Context) (Tx, error)
}

type Transaction interface {
	Executor
	Commit(context.Context) error
	Rollback(context.Context) error
}

func Exec(ctx context.Context, exec Executor, q Query) (sql.Result, error) {
	var err error

	if h, ok := q.(HookableQuery); ok {
		ctx, err = h.RunHooks(ctx, exec)
		if err != nil {
			return nil, err
		}
	}

	sql, args, err := Build(ctx, q)
	if err != nil {
		return nil, err
	}

	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	if l, ok := q.(Loadable); ok {
		for _, loader := range l.GetLoaders() {
			if err := loader.Load(ctx, exec, nil); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func One[T any](ctx context.Context, exec Executor, q Query, m scan.Mapper[T]) (T, error) {
	var t T
	var err error

	if h, ok := q.(HookableQuery); ok {
		ctx, err = h.RunHooks(ctx, exec)
		if err != nil {
			return t, err
		}
	}

	sql, args, err := Build(ctx, q)
	if err != nil {
		return t, err
	}

	if l, ok := q.(MapperModder); ok {
		if loaders := l.GetMapperMods(); len(loaders) > 0 {
			m = scan.Mod(m, loaders...)
		}
	}

	t, err = scan.One(ctx, exec, m, sql, args...)
	if err != nil {
		return t, err
	}

	if l, ok := q.(Loadable); ok {
		for _, loader := range l.GetLoaders() {
			if err := loader.Load(ctx, exec, t); err != nil {
				return t, err
			}
		}
	}

	if h, ok := any(t).(HookableType); ok {
		if err = h.AfterQueryHook(ctx, exec, q.Type()); err != nil {
			return t, err
		}
	}

	return t, err
}

func All[T any](ctx context.Context, exec Executor, q Query, m scan.Mapper[T]) ([]T, error) {
	return Allx[SliceTransformer[T, []T]](ctx, exec, q, m)
}

// SliceTransformer is a [Transformer] that transforms a scanned slice of type T into a slice of type Ts.
type SliceTransformer[T any, Ts ~[]T] struct{}

func (SliceTransformer[T, Ts]) TransformScanned(scanned []T) (Ts, error) {
	return Ts(scanned), nil
}

type Transformer[T any, V any] interface {
	TransformScanned([]T) (V, error)
}

// Allx in addition to the [scan.Mapper], Allx takes a [Transformer] that will transform the scanned slice into a different type.
// For common use cases, you can use [SliceTransformer] to transform a scanned slice of type T into a custom slice type like ~[]T.
func Allx[Tr Transformer[T, V], T, V any](ctx context.Context, exec Executor, q Query, m scan.Mapper[T]) (V, error) {
	var typedSlice V
	var err error

	_, isTransformedHookable := any(typedSlice).(HookableType)
	_, isSingleHookable := any(*new(T)).(HookableType)
	// If the transformed type is not hookable, but the single type is,
	// return an error
	if !isTransformedHookable && isSingleHookable {
		return typedSlice, ErrHookableTypeMismatch
	}

	if h, ok := q.(HookableQuery); ok {
		ctx, err = h.RunHooks(ctx, exec)
		if err != nil {
			return typedSlice, err
		}
	}

	sql, args, err := Build(ctx, q)
	if err != nil {
		return typedSlice, err
	}

	if l, ok := q.(MapperModder); ok {
		if loaders := l.GetMapperMods(); len(loaders) > 0 {
			m = scan.Mod(m, loaders...)
		}
	}

	rawSlice, err := scan.All(ctx, exec, m, sql, args...)
	if err != nil {
		return typedSlice, err
	}

	var transformer Tr
	typedSlice, err = transformer.TransformScanned(rawSlice)
	if err != nil {
		return typedSlice, err
	}

	if l, ok := q.(Loadable); ok {
		for _, loader := range l.GetLoaders() {
			if err := loader.Load(ctx, exec, typedSlice); err != nil {
				return typedSlice, err
			}
		}
	}

	if h, ok := any(typedSlice).(HookableType); ok {
		if err = h.AfterQueryHook(ctx, exec, q.Type()); err != nil {
			return typedSlice, err
		}
	}

	return typedSlice, nil
}

// Cursor returns a cursor that works similar to *sql.Rows
func Cursor[T any](ctx context.Context, exec Executor, q Query, m scan.Mapper[T]) (scan.ICursor[T], error) {
	var err error

	if h, ok := q.(HookableQuery); ok {
		ctx, err = h.RunHooks(ctx, exec)
		if err != nil {
			return nil, err
		}
	}

	sql, args, err := Build(ctx, q)
	if err != nil {
		return nil, err
	}

	if l, ok := q.(MapperModder); ok {
		if loaders := l.GetMapperMods(); len(loaders) > 0 {
			m = scan.Mod(m, loaders...)
		}
	}

	l, isLoadable := q.(Loadable)
	_, isHookable := any(*new(T)).(HookableType)

	m2 := scan.Mapper[T](func(ctx context.Context, c []string) (scan.BeforeFunc, func(any) (T, error)) {
		before, after := m(ctx, c)
		return before, func(link any) (T, error) {
			t, err := after(link)
			if err != nil {
				return t, err
			}

			if isLoadable {
				for _, loader := range l.GetLoaders() {
					err = loader.Load(ctx, exec, t)
					if err != nil {
						return t, err
					}
				}
			}

			if isHookable {
				if err = any(t).(HookableType).AfterQueryHook(ctx, exec, q.Type()); err != nil {
					return t, err
				}
			}

			return t, err
		}
	})

	return scan.Cursor(ctx, exec, m2, sql, args...)
}
