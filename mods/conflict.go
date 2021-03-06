package mods

import (
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

type Conflict[Q interface{ SetConflict(clause.Conflict) }] func() clause.Conflict

func (s Conflict[Q]) Apply(q Q) {
	q.SetConflict(s())
}

func (c Conflict[Q]) On(target any, where ...any) Conflict[Q] {
	conflict := c()
	conflict.Target.Target = target
	conflict.Target.Where = append(conflict.Target.Where, where...)

	return Conflict[Q](func() clause.Conflict {
		return conflict
	})
}

func (c Conflict[Q]) DoNothing() query.Mod[Q] {
	conflict := c()
	conflict.Do = "NOTHING"

	return Conflict[Q](func() clause.Conflict {
		return conflict
	})
}

func (c Conflict[Q]) DoUpdate() Conflict[Q] {
	conflict := c()
	conflict.Do = "UPDATE"

	return Conflict[Q](func() clause.Conflict {
		return conflict
	})
}

func (c Conflict[Q]) Set(a, b any) Conflict[Q] {
	conflict := c()
	conflict.Set.Set = append(conflict.Set.Set, expr.OP("=", a, b))

	return Conflict[Q](func() clause.Conflict {
		return conflict
	})
}

func (c Conflict[Q]) Where(where ...any) Conflict[Q] {
	conflict := c()
	conflict.Where.Conditions = append(conflict.Where.Conditions, where...)

	return Conflict[Q](func() clause.Conflict {
		return conflict
	})
}
