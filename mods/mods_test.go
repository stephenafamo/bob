package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

var (
	_ bob.Mod[any]                                 = QueryMods[any](nil)
	_ bob.Mod[interface{ AppendWith(clause.CTE) }] = With[interface{ AppendWith(clause.CTE) }]{}
)
