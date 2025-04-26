package mods

import (
	"github.com/stephenafamo/bob"
)

var _ bob.Mod[interface{ SetConflict(bob.Expression) }] = Conflict[interface{ SetConflict(bob.Expression) }](nil)
