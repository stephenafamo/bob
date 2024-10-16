package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

var _ bob.Mod[interface{ SetConflict(clause.Conflict) }] = Conflict[interface{ SetConflict(clause.Conflict) }](nil)
