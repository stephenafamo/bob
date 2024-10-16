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
type Load struct {
	loadFuncs         []Loader
	preloadMapperMods []scan.MapperMod
}

func (l *Load) SetMapperMods(mods ...scan.MapperMod) {
	l.preloadMapperMods = mods
}

// GetMapperMods implements the [MapperModder] interface
func (l *Load) GetMapperMods() []scan.MapperMod {
	return l.preloadMapperMods
}

// AppendMapperMod adds to the query's mapper mods
func (l *Load) AppendMapperMod(f scan.MapperMod) {
	l.preloadMapperMods = append(l.preloadMapperMods, f)
}

// SetLoaders sets the query's loaders
func (l *Load) SetLoaders(loaders ...Loader) {
	l.loadFuncs = loaders
}

// GetLoaders implements the [Loadable] interface
func (l *Load) GetLoaders() []Loader {
	return l.loadFuncs
}

// AppendLoader add to the query's loaders
func (l *Load) AppendLoader(f ...Loader) {
	l.loadFuncs = append(l.loadFuncs, f...)
}
