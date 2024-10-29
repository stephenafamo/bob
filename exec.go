package bob

import (
	"context"
	"database/sql"

	"github.com/stephenafamo/scan"
)

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
	return Allx[T, []T](ctx, exec, q, m)
}

// Allx takes 2 type parameters. The second is a special return type of the returned slice
// this is especially useful for when the the [Query] is [Loadable] and the loader depends on the
// return value implementing an interface
func Allx[T any, Ts ~[]T](ctx context.Context, exec Executor, q Query, m scan.Mapper[T]) (Ts, error) {
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

	rawSlice, err := scan.All(ctx, exec, m, sql, args...)
	if err != nil {
		return nil, err
	}

	typedSlice := Ts(rawSlice)

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
	} else if _, ok := any(*new(T)).(HookableType); ok {
		for _, t := range typedSlice {
			if err = any(t).(HookableType).AfterQueryHook(ctx, exec, q.Type()); err != nil {
				return typedSlice, err
			}
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
