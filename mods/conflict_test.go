package mods

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/clause"
)

var _ bob.Mod[interface{ SetConflict(clause.Conflict) }] = Conflict[interface{ SetConflict(clause.Conflict) }](nil)
