package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

var (
	_ bob.Mod[any]                                = QueryMods[any](nil)
	_ bob.Mod[interface{ AppendCTE(clause.CTE) }] = With[interface{ AppendCTE(clause.CTE) }]{}
)
