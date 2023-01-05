package psql

import (
	"context"
	"database/sql"
	"fmt"
	"hash/maphash"
	"math/rand"
	"reflect"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/select/qm"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

//nolint:gochecknoglobals,gosec
var randsrc = rand.New(rand.NewSource(int64(new(maphash.Hash).Sum64())))

type Loader func(ctx context.Context, exec bob.Executor, retrieved any) error

func (l Loader) Load(ctx context.Context, exec bob.Executor, retrieved any) error {
	return l(ctx, exec, retrieved)
}

func (l Loader) Apply(q *dialect.SelectQuery) {
	q.AppendLoader(l)
}

func (l Loader) modifyPreloader(s *preloadSettings) {
	s.extraLoader.AppendLoader(l)
}

// Preloader must be a preload option to be able to have subloaders
var _ PreloadOption = Preloader(nil)

type Preloader func(ctx context.Context) (bob.Mod[*dialect.SelectQuery], scan.MapperMod, []bob.Loader)

func (l Preloader) Apply(q *dialect.SelectQuery) {
	m, f, exl := l(context.Background()) // top level pre load has blank context

	q.AppendPreloadMod(m)
	q.AppendMapperMod(f)
	q.AppendLoader(exl...)
}

func (l Preloader) modifyPreloader(s *preloadSettings) {
	s.subLoaders = append(s.subLoaders, l)
}

type canPreload interface {
	Preload(name string, rel any) error
}

func newPreloadSettings[T any, Ts ~[]T](cols []string) preloadSettings {
	return preloadSettings{
		columns:     cols,
		extraLoader: internal.NewAfterPreloader[T, Ts](),
	}
}

type preloadSettings struct {
	columns     []string
	subLoaders  []Preloader
	extraLoader *internal.AfterPreloader
}

type PreloadOption interface {
	modifyPreloader(*preloadSettings)
}

type onlyColumnsOpt []string

func (c onlyColumnsOpt) modifyPreloader(el *preloadSettings) {
	if len(c) > 0 {
		el.columns = orm.Only(el.columns, c...)
	}
}

type exceptColumnsOpt []string

func (c exceptColumnsOpt) modifyPreloader(el *preloadSettings) {
	if len(c) > 0 {
		el.columns = orm.Except(el.columns, c...)
	}
}

func LoadOnlyColumns(cols ...string) onlyColumnsOpt {
	return onlyColumnsOpt(cols)
}

func LoadExceptColumns(cols ...string) exceptColumnsOpt {
	return exceptColumnsOpt(cols)
}

func Preload[T any, Ts ~[]T](rel orm.Relationship, cols []string, opts ...PreloadOption) Preloader {
	settings := newPreloadSettings[T, Ts](cols)
	for _, o := range opts {
		if o == nil {
			continue
		}
		o.modifyPreloader(&settings)
	}

	return preloader[T](func(ctx context.Context) (string, mods.QueryMods[*dialect.SelectQuery]) {
		parent, _ := ctx.Value(orm.CtxLoadParentAlias).(string)
		if parent == "" {
			parent = rel.Sides[0].From
		}

		var alias string
		var queryMods mods.QueryMods[*dialect.SelectQuery]

		for _, side := range rel.Sides {
			alias = fmt.Sprintf("%s_%d", side.To, randsrc.Int63n(10000))
			on := make([]any, 0, len(side.FromColumns)+len(side.FromWhere)+len(side.ToWhere))
			for i, fromCol := range side.FromColumns {
				toCol := side.ToColumns[i]
				on = append(on, X(
					Quote(parent, fromCol),
					"=",
					Quote(alias, toCol),
				))
			}
			for _, from := range side.FromWhere {
				on = append(on, X(
					Quote(parent, from.Column),
					"=",
					from.Value,
				))
			}
			for _, to := range side.ToWhere {
				on = append(on, X(
					Quote(alias, to.Column),
					"=",
					to.Value,
				))
			}

			queryMods = append(queryMods, qm.
				LeftJoin(Quote(side.To)).
				As(alias).
				On(on...))
		}

		queryMods = append(queryMods, qm.Columns(
			orm.NewColumns(settings.columns...).WithParent(alias).WithPrefix(alias+"."),
		))
		return alias, queryMods
	}, rel.Name, settings)
}

func preloader[T any](f func(context.Context) (string, mods.QueryMods[*dialect.SelectQuery]), name string, opt preloadSettings) Preloader {
	return func(ctx context.Context) (bob.Mod[*dialect.SelectQuery], scan.MapperMod, []bob.Loader) {
		alias, queryMods := f(ctx)
		prefix := alias + "."

		var mapperMods []scan.MapperMod
		extraLoaders := []bob.Loader{opt.extraLoader}

		ctx = context.WithValue(ctx, orm.CtxLoadParentAlias, alias)
		for _, l := range opt.subLoaders {
			queryMods = append(queryMods, l)
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
				loader, isLoader := retrieved.(canPreload)
				if !isLoader {
					return fmt.Errorf("object %T cannot pre load", retrieved)
				}

				t, err := after(link)
				if err != nil {
					return err
				}

				if err = opt.extraLoader.Collect(t); err != nil {
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
