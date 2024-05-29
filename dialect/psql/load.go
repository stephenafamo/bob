package psql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// Loader builds a query mod that makes an extra query after the object is retrieved
// it can be used to prevent N+1 queries by loading relationships in batches
type Loader = internal.Loader[*dialect.SelectQuery]

// Preloader builds a query mod that modifies the original query to retrieve related fields
// while it can be used as a queryMod, it does not have any direct effect.
// if using manually, the ApplyPreload method should be called
// with the query's context AFTER other mods have been applied
type Preloader = internal.Preloader[*dialect.SelectQuery]

// Settings for preloading relationships
type PreloadSettings = internal.PreloadSettings[*dialect.SelectQuery]

// Modifies preloading relationships
type PreloadOption = internal.PreloadOption[*dialect.SelectQuery]

func PreloadOnly(cols ...string) PreloadOption {
	return internal.PreloadOnly[*dialect.SelectQuery](cols)
}

func PreloadExcept(cols ...string) PreloadOption {
	return internal.PreloadExcept[*dialect.SelectQuery](cols)
}

func PreloadWhere(f ...func(from, to string) []bob.Expression) PreloadOption {
	return internal.PreloadWhere[*dialect.SelectQuery](f)
}

func PreloadAs(alias string) PreloadOption {
	return internal.PreloadAs[*dialect.SelectQuery](alias)
}

func Preload[T any, Ts ~[]T](rel orm.Relationship, cols []string, opts ...PreloadOption) Preloader {
	settings := internal.NewPreloadSettings[T, Ts, *dialect.SelectQuery](cols)
	for _, o := range opts {
		if o == nil {
			continue
		}
		o.ModifyPreloadSettings(&settings)
	}

	return buildPreloader[T](func(ctx context.Context) (string, mods.QueryMods[*dialect.SelectQuery]) {
		parent, _ := ctx.Value(orm.CtxLoadParentAlias).(string)
		if parent == "" {
			parent = rel.Sides[0].From
		}

		var alias string
		var queryMods mods.QueryMods[*dialect.SelectQuery]

		for i, side := range rel.Sides {
			alias = settings.Alias
			if settings.Alias == "" {
				alias = fmt.Sprintf("%s_%d", side.To, internal.RandInt())
			}
			on := make([]bob.Expression, 0, len(side.FromColumns)+len(side.FromWhere)+len(side.ToWhere))
			for i, fromCol := range side.FromColumns {
				toCol := side.ToColumns[i]
				on = append(on, Quote(parent, fromCol).EQ(Quote(alias, toCol)))
			}
			for _, from := range side.FromWhere {
				on = append(on, Quote(parent, from.Column).EQ(Raw(from.SQLValue)))
			}
			for _, to := range side.ToWhere {
				on = append(on, Quote(alias, to.Column).EQ(Raw(to.SQLValue)))
			}

			if len(settings.Mods) > i {
				for _, additional := range settings.Mods[i] {
					on = append(on, additional(parent, alias)...)
				}
			}

			queryMods = append(queryMods, sm.
				LeftJoin(side.ToExpr(ctx)).
				As(alias).
				On(on...))

			// so the condition on the next "side" will be on the right table
			parent = alias
		}

		queryMods = append(queryMods, mods.Preload[*dialect.SelectQuery]{
			orm.NewColumns(settings.Columns...).WithParent(alias).WithPrefix(alias + "."),
		})
		return alias, queryMods
	}, rel.Name, settings)
}

func buildPreloader[T any](f func(context.Context) (string, mods.QueryMods[*dialect.SelectQuery]), name string, opt PreloadSettings) Preloader {
	return func(ctx context.Context) (bob.Mod[*dialect.SelectQuery], scan.MapperMod, []bob.Loader) {
		alias, queryMods := f(ctx)
		prefix := alias + "."

		var mapperMods []scan.MapperMod
		extraLoaders := []bob.Loader{opt.ExtraLoader}

		ctx = context.WithValue(ctx, orm.CtxLoadParentAlias, alias)
		for _, l := range opt.SubLoaders {
			queryMod, mapperMod, extraLoader := l(ctx)
			if queryMod != nil {
				queryMods = append(queryMods, queryMod)
			}

			if mapperMod != nil {
				mapperMods = append(mapperMods, mapperMod)
			}

			if extraLoader != nil {
				extraLoaders = append(extraLoaders, extraLoader...)
			}
		}

		return queryMods, func(ctx context.Context, cols []string) (scan.BeforeFunc, scan.AfterMod) {
			before, after := scan.StructMapper[T](
				scan.WithStructTagPrefix(prefix),
				scan.WithTypeConverter(typeConverter{}),
				scan.WithRowValidator(rowValidator),
				scan.WithMapperMods(mapperMods...),
			)(ctx, cols)

			return before, func(link, retrieved any) error {
				loader, isLoader := retrieved.(internal.Preloadable)
				if !isLoader {
					return fmt.Errorf("object %T cannot pre load", retrieved)
				}

				t, err := after(link)
				if err != nil {
					return err
				}

				if err = opt.ExtraLoader.Collect(t); err != nil {
					return err
				}

				return loader.Preload(name, t)
			}
		}, extraLoaders
	}
}

// the row is valid if at least one column is not null
func rowValidator(_ []string, vals []reflect.Value) bool {
	for _, v := range vals {
		v, ok := v.Interface().(*wrapper)
		if !ok {
			return false
		}

		if !v.IsNull {
			return true
		}
	}

	return false
}

type wrapper struct {
	IsNull bool
	V      any
}

// Scan implements the sql.Scanner interface. If the wrapped type implements
// sql.Scanner then it will call that.
func (v *wrapper) Scan(value any) error {
	if value == nil {
		v.IsNull = true
		return nil
	}

	if scanner, ok := v.V.(sql.Scanner); ok {
		return scanner.Scan(value)
	}

	return opt.ConvertAssign(v.V, value)
}

type typeConverter struct{}

func (typeConverter) TypeToDestination(typ reflect.Type) reflect.Value {
	val := reflect.ValueOf(&wrapper{
		V: reflect.New(typ).Interface(),
	})

	return val
}

func (typeConverter) ValueFromDestination(val reflect.Value) reflect.Value {
	return val.Elem().FieldByName("V").Elem().Elem()
}
