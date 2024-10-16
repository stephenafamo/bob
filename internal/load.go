package internal

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type Preloadable interface {
	Preload(name string, rel any) error
}

type loadable interface {
	AppendLoader(f ...bob.Loader)
	AppendMapperMod(f scan.MapperMod)
}

// Loader builds a query mod that makes an extra query after the object is retrieved
// it can be used to prevent N+1 queries by loading relationships in batches
type Loader[Q loadable] func(ctx context.Context, exec bob.Executor, retrieved any) error

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

func NewPreloadSettings[T any, Ts ~[]T, Q loadable](cols []string) PreloadSettings[Q] {
	return PreloadSettings[Q]{
		Columns:     cols,
		ExtraLoader: NewAfterPreloader[T, Ts](),
	}
}

type preloadfilter = func(from, to string) []bob.Expression

type PreloadSettings[Q loadable] struct {
	Columns     []string
	SubLoaders  []Preloader[Q]
	ExtraLoader *AfterPreloader
	Mods        [][]preloadfilter
	Alias       string
}

type PreloadOption[Q loadable] interface {
	ModifyPreloadSettings(*PreloadSettings[Q])
}

type PreloadAs[Q loadable] string

func (o PreloadAs[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	el.Alias = string(o)
}

type PreloadOnly[Q loadable] []string

func (o PreloadOnly[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	if len(o) > 0 {
		el.Columns = orm.Only(el.Columns, o...)
	}
}

type PreloadExcept[Q loadable] []string

func (e PreloadExcept[Q]) ModifyPreloadSettings(el *PreloadSettings[Q]) {
	if len(e) > 0 {
		el.Columns = orm.Except(el.Columns, e...)
	}
}

type PreloadWhere[Q loadable] []preloadfilter

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
type Preloader[Q loadable] func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader)

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
		return fmt.Errorf("Expected to receive %s but got %T", a.oneType.String(), v)
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
