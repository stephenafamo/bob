package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

// To be embedded in other query mod providers
type FromMod[Q interface{ AppendFromItem(clause.FromItem) }] struct{}

func (FromMod[Q]) From(table any, fromMods ...bob.Mod[*clause.FromItem]) bob.Mod[Q] {
	f := clause.FromItem{}

	switch t := table.(type) {
	case string:
		f.Table = t // early because it is a common case
	case bob.Query:
		f.Table = expr.P(table) // wrap in brackets
	case bob.Mod[*clause.FromItem]:
		fromMods = append([]bob.Mod[*clause.FromItem]{t}, fromMods...)
	default:
		f.Table = t
	}

	for _, mod := range fromMods {
		mod.Apply(&f)
	}

	return FromItems[Q](f)
}

type TableAliasMod[Q interface{ SetTableAlias(string, ...string) }] struct{}

func (TableAliasMod[Q]) As(alias string, columns ...string) bob.Mod[Q] {
	return TableAs[Q]{
		Alias:   alias,
		Columns: columns,
	}
}
