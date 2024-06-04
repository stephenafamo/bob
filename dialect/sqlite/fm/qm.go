package fm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func Distinct() bob.Mod[*dialect.Function] {
	return mods.QueryModFunc[*dialect.Function](func(f *dialect.Function) {
		f.Distinct = true
	})
}

func OrderBy(e any) dialect.OrderBy[*dialect.Function] {
	return dialect.OrderBy[*dialect.Function](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Filter(e ...any) bob.Mod[*dialect.Function] {
	return mods.QueryModFunc[*dialect.Function](func(f *dialect.Function) {
		f.Filter = append(f.Filter, e...)
	})
}

func Over() dialect.WindowMod[*dialect.Function] {
	m := dialect.WindowMod[*dialect.Function]{}
	m.WindowChain = &dialect.WindowChain[*dialect.WindowMod[*dialect.Function]]{
		Wrap: &m,
	}
	return m
}
