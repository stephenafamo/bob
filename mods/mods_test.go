package mods

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/clause"
)

var (
	_ bob.Mod[any]                                 = QueryMods[any](nil)
	_ bob.Mod[interface{ AppendWith(clause.CTE) }] = With[interface{ AppendWith(clause.CTE) }]{}
)
