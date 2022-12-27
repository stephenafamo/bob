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
	LoadFuncs         []Loader
	PreloadMapperMods []scan.MapperMod
	PreloadMods       []Mod[Q]
}

// GetMapperMods implements the [MapperModder] interface
func (l *Load[Q]) GetMapperMods() []scan.MapperMod {
	return l.PreloadMapperMods
}

// AppendMapperMod adds to the query's mapper mods
func (l *Load[Q]) AppendMapperMod(f scan.MapperMod) {
	l.PreloadMapperMods = append(l.PreloadMapperMods, f)
}

// GetLoaders implements the [Loadable] interface
func (l *Load[Q]) GetLoaders() []Loader {
	return l.LoadFuncs
}

// AppendLoader add to the query's loaders
func (l *Load[Q]) AppendLoader(f ...Loader) {
	l.LoadFuncs = append(l.LoadFuncs, f...)
}

// AppendPreloadMod adds a preload mod to the query
// PreloadMods are applied just before expressing
func (l *Load[Q]) AppendPreloadMod(m Mod[Q]) {
	l.PreloadMods = append(l.PreloadMods, m)
}
