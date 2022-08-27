package bob

import (
	"context"

	"github.com/stephenafamo/scan"
)

type (
	MapperModder interface {
		GetMapperMods() []scan.MapperMod
	}

	LoadFunc = func(ctx context.Context, exec scan.Queryer, retrieved any) error
	Loadable interface {
		GetLoaders() []LoadFunc
		GetExtraLoaders() []ExtraLoader
	}

	ExtraLoader interface {
		LoadOne(context.Context, scan.Queryer) error
		LoadMany(context.Context, scan.Queryer) error
	}

	ExecSettings[T any] struct {
		AfterSelect func(ctx context.Context, retrieved []T) error
	}

	ExecOption[T any] func(*ExecSettings[T])
)

func One[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (T, error) {
	settings := ExecSettings[T]{}
	for _, opt := range opts {
		opt(&settings)
	}

	var t T

	sql, args, err := Build(q)
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
			if err := loader(ctx, exec, t); err != nil {
				return t, err
			}
		}
		for _, loader := range l.GetExtraLoaders() {
			if err := loader.LoadOne(ctx, exec); err != nil {
				return t, err
			}
		}
	}

	if settings.AfterSelect != nil {
		if err := settings.AfterSelect(ctx, []T{t}); err != nil {
			return t, err
		}
	}

	return t, err
}

func All[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) ([]T, error) {
	return Allx[T, []T](ctx, exec, q, m, opts...)
}

// Allx takes 2 type parameters. The second is a special return type of the returned slice
// this is especially useful for when the the [Query] is [Loadable] and the loader depends on the
// return value implementing an interface
func Allx[T any, Ts ~[]T](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (Ts, error) {
	settings := ExecSettings[T]{}
	for _, opt := range opts {
		opt(&settings)
	}

	sql, args, err := Build(q)
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
			if err := loader(ctx, exec, typedSlice); err != nil {
				return typedSlice, err
			}
		}
		for _, loader := range l.GetExtraLoaders() {
			if err := loader.LoadMany(ctx, exec); err != nil {
				return typedSlice, err
			}
		}
	}

	if settings.AfterSelect != nil {
		if err := settings.AfterSelect(ctx, typedSlice); err != nil {
			return typedSlice, err
		}
	}

	return typedSlice, nil
}

// Cursor returns a cursor that works similar to *sql.Rows
func Cursor[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T], opts ...ExecOption[T]) (scan.ICursor[T], error) {
	settings := ExecSettings[T]{}
	for _, opt := range opts {
		opt(&settings)
	}

	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	if l, ok := q.(MapperModder); ok {
		if loaders := l.GetMapperMods(); len(loaders) > 0 {
			m = scan.Mod(m, loaders...)
		}
	}

	l, ok := q.(Loadable)
	if !ok {
		return scan.Cursor(ctx, exec, m, sql, args...)
	}

	m2 := scan.Mapper[T](func(ctx context.Context, c map[string]int) func(*scan.Values) (T, error) {
		mapFunc := m(ctx, c)
		return func(v *scan.Values) (T, error) {
			t, err := mapFunc(v)
			if err != nil {
				return t, err
			}

			for _, loader := range l.GetLoaders() {
				err = loader(ctx, exec, t)
				if err != nil {
					return t, err
				}
			}
			for _, loader := range l.GetExtraLoaders() {
				if err := loader.LoadOne(ctx, exec); err != nil {
					return t, err
				}
			}

			if settings.AfterSelect != nil {
				if err := settings.AfterSelect(ctx, []T{t}); err != nil {
					return t, err
				}
			}
			return t, err
		}
	})

	return scan.Cursor(ctx, exec, m2, sql, args...)
}
