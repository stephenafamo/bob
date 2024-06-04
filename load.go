package bob

import (
	"context"

	"github.com/stephenafamo/scan"
)

type (
	// Loadable is an object that has loaders
	// if a query implements this interface, the loaders are called
	// after executing the query
	Loadable interface {
		GetLoaders() []Loader
	}

	// Loader is an object that is called after the main query is performed
	// when called from [Exec], retrieved is nil
	// when called from [One], retrieved is the retrieved object
	// when called from [All], retrieved is a slice retrieved objects
	// this is used for loading relationships
	Loader interface {
		Load(ctx context.Context, exec Executor, retrieved any) error
	}
)

// Load is an embeddable struct that enables Preloading and AfterLoading
type Load[Q any] struct {
	loadContext       context.Context
	loadFuncs         []Loader
	preloadMapperMods []scan.MapperMod
}

// GetLoadContext
func (l *Load[Q]) GetLoadContext() context.Context {
	return l.loadContext
}

// SetLoadContext
func (l *Load[Q]) SetLoadContext(ctx context.Context) {
	l.loadContext = ctx
}

func (l *Load[Q]) SetMapperMods(mods ...scan.MapperMod) {
	l.preloadMapperMods = mods
}

// GetMapperMods implements the [MapperModder] interface
func (l *Load[Q]) GetMapperMods() []scan.MapperMod {
	return l.preloadMapperMods
}

// AppendMapperMod adds to the query's mapper mods
func (l *Load[Q]) AppendMapperMod(f scan.MapperMod) {
	l.preloadMapperMods = append(l.preloadMapperMods, f)
}

// SetLoaders sets the query's loaders
func (l *Load[Q]) SetLoaders(loaders ...Loader) {
	l.loadFuncs = loaders
}

// GetLoaders implements the [Loadable] interface
func (l *Load[Q]) GetLoaders() []Loader {
	return l.loadFuncs
}

// AppendLoader add to the query's loaders
func (l *Load[Q]) AppendLoader(f ...Loader) {
	l.loadFuncs = append(l.loadFuncs, f...)
}
