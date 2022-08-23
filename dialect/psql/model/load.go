package model

import (
	"context"
	"database/sql"
	"fmt"
	"hash/maphash"
	"math/rand"
	"reflect"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

//nolint:gochecknoglobals
var randsrc = rand.New(rand.NewSource(int64(new(maphash.Hash).Sum64())))

type Loader bob.LoadFunc

func (l Loader) Apply(q *psql.SelectQuery) {
	q.AppendLoader(l)
}

func (l Loader) ModifyEagerLoader(s *eagerLoadSettings) {
	s.extraLoader.AppendLoader(l)
}

type EagerLoader func(ctx context.Context) (bob.Mod[*psql.SelectQuery], scan.MapperMod, []bob.ExtraLoader)

func (l EagerLoader) Apply(q *psql.SelectQuery) {
	m, f, exl := l(context.Background()) // top level eager load has blank context
	q.AppendEagerLoadMod(m)
	q.AppendMapperMod(f)
	q.AppendExtraLoader(exl...)
}

func (l EagerLoader) ModifyEagerLoader(s *eagerLoadSettings) {
	s.subLoaders = append(s.subLoaders, l)
}

type canEagerLoad interface {
	EagerLoad(name string, rel any) error
}

func NewEagerLoadSettings[T any, Ts ~[]T](cols orm.Columns) eagerLoadSettings {
	var one T
	var slice Ts
	return eagerLoadSettings{
		columns: cols,
		extraLoader: &orm.ExtraLoader{
			OneType:   reflect.TypeOf(one),
			SliceType: reflect.TypeOf(slice),
		},
	}
}

type eagerLoadSettings struct {
	columns     orm.Columns
	subLoaders  []EagerLoader
	extraLoader *orm.ExtraLoader
}

type EagerLoadOption interface {
	ModifyEagerLoader(*eagerLoadSettings)
}

type onlyColumnsOpt []string

func (c onlyColumnsOpt) ModifyEagerLoader(el *eagerLoadSettings) {
	if len(c) > 0 {
		el.columns = el.columns.Only(c...)
	}
}

type exceptColumnsOpt []string

func (c exceptColumnsOpt) ModifyEagerLoader(el *eagerLoadSettings) {
	if len(c) > 0 {
		el.columns = el.columns.Except(c...)
	}
}

func LoadOnlyColumns(cols ...string) onlyColumnsOpt {
	return onlyColumnsOpt(cols)
}

func LoadExceptColumns(cols ...string) exceptColumnsOpt {
	return exceptColumnsOpt(cols)
}

func Preload[T any, Ts ~[]T](rel orm.Relationship, cols orm.Columns, opts ...EagerLoadOption) EagerLoader {
	settings := NewEagerLoadSettings[T, Ts](cols)
	for _, o := range opts {
		if o == nil {
			continue
		}
		o.ModifyEagerLoader(&settings)
	}

	return eagerLoader[T](func(ctx context.Context) (string, mods.QueryMods[*psql.SelectQuery]) {
		alias := fmt.Sprintf("%s_%d", rel.ForeignTable, randsrc.Int63n(10000))
		parent, _ := ctx.Value(orm.CtxLoadParentAlias).(string)
		if parent == "" {
			parent = rel.LocalTable
		}

		colsMod := psql.SelectQM.Columns(settings.columns.WithParent(alias).WithPrefix(alias + "."))

		on := make([]any, 0, len(rel.ColumnPairs))
		for localCol, foreignCol := range rel.ColumnPairs {
			on = append(on, psql.X(
				psql.Quote(alias, foreignCol),
				"=",
				psql.Quote(parent, localCol),
			))
		}

		joinMod := psql.SelectQM.
			LeftJoin(psql.Quote(rel.ForeignTable)).
			As(alias).
			On(on...)

		return alias, mods.QueryMods[*psql.SelectQuery]{colsMod, joinMod}
	}, rel.Name, settings)
}

func eagerLoader[T any](f func(context.Context) (string, mods.QueryMods[*psql.SelectQuery]), name string, opt eagerLoadSettings) EagerLoader {
	return func(ctx context.Context) (bob.Mod[*psql.SelectQuery], scan.MapperMod, []bob.ExtraLoader) {
		alias, queryMods := f(ctx)
		prefix := alias + "."

		var mapperMods []scan.MapperMod
		extraLoaders := []bob.ExtraLoader{opt.extraLoader}

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

		return queryMods, func(ctx context.Context, cols map[string]int) func(v *scan.Values, retrieved any) error {
			f1 := scan.StructMapper[T](
				scan.WithStructTagPrefix(prefix),
				scan.WithTypeConverter(typeConverter{}),
				scan.WithRowValidator(rowValidator),
			)(ctx, cols)

			fs := make([]scan.MapperModFunc, len(mapperMods))
			for i, m := range mapperMods {
				fs[i] = m(ctx, cols)
			}

			return func(v *scan.Values, retrieved any) error {
				loader, isLoader := retrieved.(canEagerLoad)
				if !isLoader {
					return fmt.Errorf("object %T cannot eager load", retrieved)
				}

				t, err := f1(v)
				if err != nil {
					return err
				}

				for _, fx := range fs {
					if err := fx(v, t); err != nil {
						return err
					}
				}

				if !v.IsRecording() && len(opt.extraLoader.Fs) > 0 {
					if err = opt.extraLoader.Collect(t); err != nil {
						return err
					}
				}

				return loader.EagerLoad(name, t)
			}
		}, extraLoaders
	}
}

var _ scan.RowValidator = rowValidator

func rowValidator(vals map[string]reflect.Value) bool {
	for _, v := range vals {
		v, ok := v.Interface().(wrapper)
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

var _ scan.TypeConverter = typeConverter{}

type typeConverter struct{}

func (d typeConverter) ConvertType(typ reflect.Type) reflect.Value {
	val := reflect.ValueOf(&wrapper{
		V: reflect.New(typ).Interface(),
	})

	return val
}

func (d typeConverter) OriginalValue(val reflect.Value) reflect.Value {
	return val.FieldByName("V").Elem().Elem()
}
