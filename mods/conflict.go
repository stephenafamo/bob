package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

type Conflict[Q interface{ SetConflict(bob.Expression) }] func() clause.ConflictClause

func (s Conflict[Q]) Apply(q Q) {
	q.SetConflict(s())
}

func (c Conflict[Q]) Where(where ...any) Conflict[Q] {
	conflict := c()
	conflict.Target.Where = append(conflict.Target.Where, where...)

	return Conflict[Q](func() clause.ConflictClause {
		return conflict
	})
}

func (c Conflict[Q]) DoNothing() bob.Mod[Q] {
	conflict := c()
	conflict.Do = "NOTHING"

	return Conflict[Q](func() clause.ConflictClause {
		return conflict
	})
}

func (c Conflict[Q]) DoUpdate(sets ...bob.Mod[*clause.ConflictClause]) bob.Mod[Q] {
	conflict := c()
	conflict.Do = "UPDATE"

	for _, set := range sets {
		set.Apply(&conflict)
	}

	return Conflict[Q](func() clause.ConflictClause {
		return conflict
	})
}
