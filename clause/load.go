package clause

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

type Load[Q any] struct {
	LoadFuncs           []bob.LoadFunc
	ExtraLoaders        []bob.ExtraLoader
	EagerLoadMapperMods []scan.MapperMod
	EagerLoadMods       []bob.Mod[Q]
}

func (l *Load[Q]) GetMapperMods() []scan.MapperMod {
	return l.EagerLoadMapperMods
}

func (l *Load[Q]) AppendMapperMod(f scan.MapperMod) {
	l.EagerLoadMapperMods = append(l.EagerLoadMapperMods, f)
}

func (l *Load[Q]) AppendEagerLoadMod(m bob.Mod[Q]) {
	l.EagerLoadMods = append(l.EagerLoadMods, m)
}

func (l *Load[Q]) GetLoaders() []bob.LoadFunc {
	return l.LoadFuncs
}

func (l *Load[Q]) AppendLoader(f bob.LoadFunc) {
	l.LoadFuncs = append(l.LoadFuncs, f)
}

func (l *Load[Q]) GetExtraLoaders() []bob.ExtraLoader {
	return l.ExtraLoaders
}

func (l *Load[Q]) AppendExtraLoader(lo ...bob.ExtraLoader) {
	l.ExtraLoaders = append(l.ExtraLoaders, lo...)
}
