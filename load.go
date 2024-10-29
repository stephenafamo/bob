package bob

import (
	"context"

	"github.com/stephenafamo/scan"
)

// Loadable is an object that has loaders
// if a query implements this interface, the loaders are called
// after executing the query
type Loadable interface {
	GetLoaders() []Loader
}

// Loader is an object that is called after the main query is performed
// when called from [Exec], retrieved is nil
// when called from [One], retrieved is the retrieved object
// when called from [All], retrieved is a slice retrieved objects
// this is used for loading relationships
type Loader interface {
	Load(ctx context.Context, exec Executor, retrieved any) error
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
