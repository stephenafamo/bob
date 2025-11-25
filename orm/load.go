package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/scan"
)

// Loadable is a constraint for types that can be loaded
type Loadable interface {
	AppendLoader(f ...bob.Loader)
	AppendMapperMod(f scan.MapperMod)
}

// Loader builds a query mod that makes an extra query after the object is retrieved
// it can be used to prevent N+1 queries by loading relationships in batches
type Loader[Q Loadable] func(ctx context.Context, exec bob.Executor, retrieved any) error

// Load is called after the original object is retrieved
func (l Loader[Q]) Load(ctx context.Context, exec bob.Executor, retrieved any) error {
	return l(ctx, exec, retrieved)
}

// Apply satisfies the bob.Mod[Q] interface
func (l Loader[Q]) Apply(q Q) {
	q.AppendLoader(l)
}

// modifyPreloader makes a Loader also work as a mod for a [Preloader]
func (l Loader[Q]) ModifyPreloadSettings(s *PreloadSettings[Q]) {
	s.ExtraLoader.AppendLoader(l)
}

func NewPreloadSettings[T any, Ts ~[]T, Q Loadable](cols []string) PreloadSettings[Q] {
	return PreloadSettings[Q]{
		Columns:     cols,
		ExtraLoader: NewAfterPreloader[T, Ts](),
	}
}

type preloadfilter = func(from, to string) []bob.Expression

type PreloadSettings[Q Loadable] struct {
	Columns     []string
	SubLoaders  []Preloader[Q]
	ExtraLoader *AfterPreloader
	Mods        [][]preloadfilter
	Alias       string
}

type PreloadOption[Q Loadable] interface {
	ModifyPreloadSettings(*PreloadSettings[Q])
}

type PreloadAs[Q Loadable] string

func (o PreloadAs[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	el.Alias = string(o)
}

type PreloadOnly[Q Loadable] []string

func (o PreloadOnly[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	if len(o) > 0 {
		el.Columns = internal.Only(el.Columns, o...)
	}
}

type PreloadExcept[Q Loadable] []string

func (e PreloadExcept[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	if len(e) > 0 {
		el.Columns = internal.Except(el.Columns, e...)
	}
}

type PreloadWhere[Q Loadable] []preloadfilter

func (filters PreloadWhere[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	diff := len(filters) - len(el.Mods)
	if diff > 0 {
		extra := make([][]preloadfilter, diff)
		el.Mods = append(el.Mods, extra...)
	}

	for i, filter := range filters {
		el.Mods[i] = append(el.Mods[i], filter)
	}
}

// Preloader builds a query mod that modifies the original query to retrieve related fields
// while it can be used as a queryMod, it does not have any direct effect.
// if using manually, the ApplyPreload method should be called
// with the query's context AFTER other mods have been applied
type Preloader[Q Loadable] func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader)

// Apply satisfies bob.Mod[*dialect.SelectQuery].
// 1. It modifies the query to join the preloading table and the extra columns to retrieve
// 2. It modifies the mapper to scan the new columns.
// 3. It calls the original object's Preload method with the loaded object
func (l Preloader[Q]) Apply(q Q) {
	mod, mapperMod, afterLoaders := l("")

	mod.Apply(q)                    // add preload columns
	q.AppendMapperMod(mapperMod)    // add mapper
	q.AppendLoader(afterLoaders...) // add the loader
}

// modifyPreloader makes a Loader also work as a mod for a [Preloader]
func (l Preloader[Q]) ModifyPreloadSettings(s *PreloadSettings[Q]) {
	s.SubLoaders = append(s.SubLoaders, l)
}

// NewAfterPreloader returns a new AfterPreloader based on the given types
func NewAfterPreloader[T any, Ts ~[]T]() *AfterPreloader {
	var one T
	var slice Ts
	return &AfterPreloader{
		oneType:   reflect.TypeOf(one),
		sliceType: reflect.TypeOf(slice),
	}
}

// AfterPreloader is embedded in a Preloader to chain loading
// whenever a preloaded object is scanned, it should be collected with the Collect method
// The loading functions should be added with AppendLoader
// later, when this object is called like any other [bob.Loader], it
// calls the appended loaders with the collected objects
type AfterPreloader struct {
	oneType   reflect.Type
	sliceType reflect.Type

	funcs     []bob.Loader
	collected []any
}

func (a *AfterPreloader) AppendLoader(fs ...bob.Loader) {
	a.funcs = append(a.funcs, fs...)
}

func (a *AfterPreloader) Collect(v any) error {
	if len(a.funcs) == 0 {
		return nil
	}

	if reflect.TypeOf(v) != a.oneType {
		return fmt.Errorf("expected to receive %s but got %T", a.oneType.String(), v)
	}

	a.collected = append(a.collected, v)
	return nil
}

func (a *AfterPreloader) Load(ctx context.Context, exec bob.Executor, _ any) error {
	if len(a.collected) == 0 || len(a.funcs) == 0 {
		return nil
	}

	obj := a.collected[0]

	if len(a.collected) > 1 {
		all := reflect.MakeSlice(a.sliceType, len(a.collected), len(a.collected))
		for k, v := range a.collected {
			all.Index(k).Set(reflect.ValueOf(v))
		}

		obj = all.Interface()
	}

	for _, f := range a.funcs {
		if err := f.Load(ctx, exec, obj); err != nil {
			return err
		}
	}

	return nil
}

type Preloadable interface {
	Preload(name string, rel any) error
}

type PreloadRel[E bob.Expression] struct {
	Name  string
	Sides []PreloadSide[E]
}

type nameable[E bob.Expression] interface {
	Name() E
	Alias() string
}

type PreloadSide[E bob.Expression] struct {
	From        nameable[E]
	To          nameable[E]
	FromColumns []string `yaml:"-"`
	ToColumns   []string `yaml:"-"`

	FromWhere []RelWhere `yaml:"from_where"`
	ToWhere   []RelWhere `yaml:"to_where"`
}

type PreloadableQuery interface {
	Loadable
	AppendJoin(clause.Join)
	AppendPreloadSelect(columns ...any)
}

func Preload[T Preloadable, Ts ~[]T, E bob.Expression, Q PreloadableQuery](rel PreloadRel[E], cols []string, opts ...PreloadOption[Q]) Preloader[Q] {
	settings := NewPreloadSettings[T, Ts, Q](cols)
	for _, o := range opts {
		if o == nil {
			continue
		}
		o.ModifyPreloadSettings(&settings)
	}

	return buildPreloader[T](func(parent string) (string, mods.QueryMods[Q]) {
		if parent == "" {
			parent = rel.Sides[0].From.Alias()
		}

		var alias string
		var queryMods mods.QueryMods[Q]

		for i, side := range rel.Sides {
			alias = settings.Alias
			if settings.Alias == "" {
				alias = fmt.Sprintf("%s_%d", side.To.Alias(), internal.RandInt())
			}
			on := make([]bob.Expression, 0, len(side.FromColumns)+len(side.FromWhere)+len(side.ToWhere))
			for i, fromCol := range side.FromColumns {
				toCol := side.ToColumns[i]
				on = append(on, expr.OP(
					"=",
					expr.Quote(parent, fromCol),
					expr.Quote(alias, toCol),
				))
			}
			for _, from := range side.FromWhere {
				on = append(on, expr.OP(
					"=",
					expr.Quote(parent, from.Column),
					expr.Raw(from.SQLValue),
				))
			}
			for _, to := range side.ToWhere {
				on = append(on, expr.OP(
					"=",
					expr.Quote(parent, to.Column),
					expr.Raw(to.SQLValue),
				))
			}

			if len(settings.Mods) > i {
				for _, additional := range settings.Mods[i] {
					on = append(on, additional(parent, alias)...)
				}
			}

			queryMods = append(queryMods, mods.Join[Q](clause.Join{
				Type: clause.LeftJoin,
				To: clause.TableRef{
					Expression: side.To.Name(),
					Alias:      alias,
				},
				On: on,
			}))

			// so the condition on the next "side" will be on the right table
			parent = alias
		}

		queryMods = append(queryMods, mods.Preload[Q]{
			expr.NewColumnsExpr(settings.Columns...).WithParent(alias).WithPrefix(alias + "."),
		})
		return alias, queryMods
	}, rel.Name, settings)
}

func buildPreloader[T any, Q Loadable](f func(string) (string, mods.QueryMods[Q]), name string, opt PreloadSettings[Q]) Preloader[Q] {
	return func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader) {
		alias, queryMods := f(parent)
		prefix := alias + "."

		var mapperMods []scan.MapperMod
		extraLoaders := []bob.Loader{opt.ExtraLoader}

		for _, l := range opt.SubLoaders {
			queryMod, mapperMod, extraLoader := l(alias)
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
				scan.WithTypeConverter(NullTypeConverter{}),
				scan.WithRowValidator(rowValidator),
				scan.WithMapperMods(mapperMods...),
			)(ctx, cols)

			return before, func(link, retrieved any) error {
				loader, isLoader := retrieved.(Preloadable)
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

// NullTypeConverter is a TypeConverter that skips NULL values during scanning even if the destination type does not support NULLs.
// This is useful when scanning complex queries with optional relationships while still wanting to re-use some generated structs.
//

// Example usage:

// Assuming the following generated type in the package "gen":
//   type Thing struct {
//     ID        string              `db:"id,pk" `
//     Name      string              `db:"name" `
//     Country   sql.Null[string]    `db:"country" `
//   }
//
// And the following custom struct that includes the generated type as a field:
//
// type myRow struct {
//     ... so many other cols
//     OptionalThing gen.Thing `db:"thing"` // will be populated by selecting thing.id, thing.name, thing.country
// }
//
// The "thing" table columns are loaded via a LEFT JOIN, so they could be all NULL: NullTypeConverter will make sure that the scan won't fail, leaving the struct empty (like Preload would do).
//
// bob.All(ctx, db,
//   psql.Select(
//      sm.Columns(
//         ...
//         gen.Things.Columns.WithPrefix("thing.")),
//      sm.From(...),
//		sm.LeftJoin(gen.Things.Name()).As("thing").On(
//			gen.Things.Columns.AliasedAs("thing").ID.EQ(...),
//		),
// 	), scan.StructMapper[myRow](scan.WithTypeConverter(orm.NullTypeConverter{})))

type NullTypeConverter struct{}

// TypeToDestination implements the TypeConverter interface and returns a reflect.Value that wraps the destination type in a wrapper struct able to handle NULL values.
func (NullTypeConverter) TypeToDestination(typ reflect.Type) reflect.Value {
	val := reflect.ValueOf(&wrapper{
		V: reflect.New(typ).Interface(),
	})

	return val
}

// ValueFromDestination implements the TypeConverter interface and extracts the actual value from the wrapper struct.
func (NullTypeConverter) ValueFromDestination(val reflect.Value) reflect.Value {
	return val.Elem().FieldByName("V").Elem().Elem()
}
