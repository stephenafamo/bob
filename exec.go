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
)

func One[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) (T, error) {
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

	return t, err
}

func All[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) ([]T, error) {
	return Allx[T, []T](ctx, exec, q, m)
}

// Allx takes 2 type parameters. The second is a special return type of the returned slice
// this is especially useful for when the the [Query] is [Loadable] and the loader depends on the
// return value implementing an interface
func Allx[T any, Ts ~[]T](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) (Ts, error) {
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

	return typedSlice, nil
}

// Cursor returns a cursor that works similar to *sql.Rows
func Cursor[T any](ctx context.Context, exec scan.Queryer, q Query, m scan.Mapper[T]) (scan.ICursor[T], error) {
	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	if l, ok := q.(MapperModder); ok {
		if loaders := l.GetMapperMods(); len(loaders) > 0 {
			m = scan.Mod(m, loaders...)
		}
	}

	if l, ok := q.(Loadable); ok {
		m2 := scan.Mapper[T](func(ctx context.Context, c map[string]int) func(*scan.Values) (T, error) {
			mapFunc := m(ctx, c)
			return func(v *scan.Values) (T, error) {
				o, err := mapFunc(v)
				if err != nil {
					return o, err
				}

				for _, v := range l.GetLoaders() {
					err = v(ctx, exec, o)
					if err != nil {
						return o, err
					}
				}

				return o, err
			}
		})
		return scan.Cursor(ctx, exec, m2, sql, args...)
	}

	return scan.Cursor(ctx, exec, m, sql, args...)
}

// Collect multiple slices of values from a single query
// collector must be of the structure
// func(cols) func(*Values) (t1, t2, ..., error)
// The returned slice contains values like this
// {[]t1, []t2}
func Collect(ctx context.Context, exec scan.Queryer, q Query, collector func(context.Context, map[string]int) any) ([]any, error) {
	sql, args, err := Build(q)
	if err != nil {
		return nil, err
	}

	return scan.Collect(ctx, exec, collector, sql, args...)
}
