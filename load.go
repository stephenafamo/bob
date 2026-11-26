package bob

import (
	"context"
	"slices"

	"github.com/stephenafamo/scan"
)

// Loadable is an object that has loaders
// if a query implements this interface, the loaders are called
// after executing the query
type Loadable interface {
	GetLoaders(context.Context) []Loader
}

// Loader is an object that is called after the main query is performed
// when called from [Exec], retrieved is nil
// when called from [One], retrieved is the retrieved object
// when called from [All], retrieved is a slice retrieved objects
// this is used for loading relationships
type Loader interface {
	Load(ctx context.Context, exec Executor, retrieved any) error
}

// LoaderContext lets a loader and its scan mapper share state for one
// execution, including when the same query or prepared statement is reused.
type LoaderContext interface {
	LoaderContext(context.Context) context.Context
}

func loaderContext(ctx context.Context, loaders []Loader) context.Context {
	for _, l := range loaders {
		if local, ok := l.(LoaderContext); ok {
			ctx = local.LoaderContext(ctx)
		}
	}
	return ctx
}

func executionLoaders(ctx context.Context, q Query) (context.Context, []Loader) {
	loaders := loadersFor(ctx, q)
	return loaderContext(ctx, loaders), loaders
}

func loadersFor(ctx context.Context, e Expression) []Loader {
	switch l := e.(type) {
	case Loadable:
		return l.GetLoaders(ctx)
	case interface{ GetLoaders() []Loader }: // Existing custom queries.
		return l.GetLoaders()
	}
	return nil
}

func mapperModsFor(ctx context.Context, e Expression) []scan.MapperMod {
	switch m := e.(type) {
	case MapperModder:
		return m.GetMapperMods(ctx)
	case interface{ GetMapperMods() []scan.MapperMod }:
		return m.GetMapperMods()
	}
	return nil
}

func queryMapper[T any](ctx context.Context, q Query, m scan.Mapper[T]) scan.Mapper[T] {
	if mods := mapperModsFor(ctx, q); len(mods) > 0 {
		return scan.Mod(m, mods...)
	}
	return m
}

// Loader builds a query mod that makes an extra query after the object is retrieved
// it can be used to prevent N+1 queries by loading relationships in batches
type LoaderFunc func(ctx context.Context, exec Executor, retrieved any) error

// Load is called after the original object is retrieved
func (l LoaderFunc) Load(ctx context.Context, exec Executor, retrieved any) error {
	return l(ctx, exec, retrieved)
}

// Load is an embeddable struct that enables Preloading and AfterLoading
type Load struct {
	loadFuncs         []Loader
	preloadMapperMods []scan.MapperMod
	uniqueCounter     uint64
}

// Clip prevents a branch from appending to the parent's callback storage.
func (l *Load) Clip() {
	l.loadFuncs = slices.Clip(l.loadFuncs)
	l.preloadMapperMods = slices.Clip(l.preloadMapperMods)
}

func (l *Load) SetMapperMods(mods ...scan.MapperMod) {
	l.preloadMapperMods = slices.Clone(mods)
}

// GetMapperMods implements the [MapperModder] interface
func (l *Load) GetMapperMods(context.Context) []scan.MapperMod {
	return l.preloadMapperMods
}

// AppendMapperMod adds to the query's mapper mods
func (l *Load) AppendMapperMod(f scan.MapperMod) {
	l.preloadMapperMods = append(l.preloadMapperMods, f)
}

// SetLoaders sets the query's loaders
func (l *Load) SetLoaders(loaders ...Loader) {
	l.loadFuncs = slices.Clone(loaders)
}

// GetLoaders implements the [Loadable] interface
func (l *Load) GetLoaders(context.Context) []Loader {
	return l.loadFuncs
}

// AppendLoader add to the query's loaders
func (l *Load) AppendLoader(f ...Loader) {
	l.loadFuncs = append(l.loadFuncs, f...)
}

// NextUniqueInt is like the package level [NextUniqueInt], but the integers are only unique
// within the query. Every query starts at 10001, so the same query always gets the same aliases.
func (l *Load) NextUniqueInt() uint64 {
	l.uniqueCounter++
	return l.uniqueCounter + 10000
}
