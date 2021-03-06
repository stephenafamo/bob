package mods

import (
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

// To be embeded in other query mod providers
type FromMod[Q interface{ AppendFromItem(clause.FromItem) }] struct{}

func (FromMod[Q]) From(table any, fromMods ...query.Mod[*clause.FromItem]) query.Mod[Q] {
	f := clause.FromItem{}

	switch t := table.(type) {
	case string:
		f.Table = t // early because it is a common case
	case query.Query:
		f.Table = expr.P(table) // wrap in brackets
	case query.Mod[*clause.FromItem]:
		fromMods = append([]query.Mod[*clause.FromItem]{t}, fromMods...)
	default:
		f.Table = t
	}

	for _, mod := range fromMods {
		mod.Apply(&f)
	}

	return FromItems[Q](f)
}

type TableAliasMod[Q interface{ SetTableAlias(string, ...string) }] struct{}

func (TableAliasMod[Q]) As(alias string, columns ...string) query.Mod[Q] {
	return TableAs[Q]{
		Alias:   alias,
		Columns: columns,
	}
}
