package mods

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

type Load[Q bob.Loadable] bob.LoadFunc

func (l Load[Q]) Apply(q Q) {
	q.AppendLoader(l)
}

type EagerLoad[Q interface {
	bob.MapperModder
	AppendEagerLoadMod(bob.Mod[Q])
}] func(ctx context.Context) (bob.Mod[Q], scan.MapperMod)

func (l EagerLoad[Q]) Apply(q Q) {
	m, f := l(context.Background()) // top level eager load has blank context
	q.AppendEagerLoadMod(m)
	q.AppendMapperMod(f)
}
