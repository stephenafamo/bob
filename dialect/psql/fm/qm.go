package fm

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/clause"
	"github.com/twitter-payments/bob/dialect/psql/dialect"
	"github.com/twitter-payments/bob/mods"
)

func Distinct() bob.Mod[*dialect.Function] {
	return bob.ModFunc[*dialect.Function](func(f *dialect.Function) {
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

func WithinGroup() bob.Mod[*dialect.Function] {
	return bob.ModFunc[*dialect.Function](func(f *dialect.Function) {
		f.WithinGroup = true
	})
}

func Filter(e ...any) bob.Mod[*dialect.Function] {
	return bob.ModFunc[*dialect.Function](func(f *dialect.Function) {
		f.Filter = append(f.Filter, e...)
	})
}

func Over(winMods ...bob.Mod[*clause.Window]) bob.Mod[*dialect.Function] {
	w := clause.Window{}
	for _, mod := range winMods {
		mod.Apply(&w)
	}

	return mods.Window[*dialect.Function](w)
}

func As(alias string) bob.Mod[*dialect.Function] {
	return bob.ModFunc[*dialect.Function](func(f *dialect.Function) {
		f.Alias = alias
	})
}

func Columns(name, datatype string) bob.Mod[*dialect.Function] {
	return bob.ModFunc[*dialect.Function](func(f *dialect.Function) {
		f.AppendColumn(name, datatype)
	})
}
