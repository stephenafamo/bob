package mods

import (
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

// To be embeded in other query mod providers
type FromMod[Q interface{ AppendFromItem(expr.FromItem) }] struct{}

func (FromMod[Q]) From(table any, fromMods ...QueryMod[*expr.FromItem]) QueryMod[Q] {
	f := expr.FromItem{}

	switch t := table.(type) {
	case string:
		f.Table = t // early because it is a common case
	case query.Query:
		f.Table = expr.P(table) // wrap in brackets
	case QueryMod[*expr.FromItem]:
		fromMods = append([]QueryMod[*expr.FromItem]{t}, fromMods...)
	default:
		f.Table = t
	}

	for _, mod := range fromMods {
		mod.Apply(&f)
	}

	return FromItems[Q](f)
}

type TableAliasMod[Q interface{ SetTableAlias(string, ...string) }] struct{}

func (TableAliasMod[Q]) As(alias string, columns ...string) QueryMod[Q] {
	return TableAs[Q]{
		Alias:   alias,
		Columns: columns,
	}
}
