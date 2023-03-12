package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

type Conflict[Q interface{ SetConflict(clause.Conflict) }] func() clause.Conflict

func (s Conflict[Q]) Apply(q Q) {
	q.SetConflict(s())
}

func (c Conflict[Q]) OnWhere(where ...any) Conflict[Q] {
	conflict := c()
	conflict.Target.Where = append(conflict.Target.Where, where...)

	return Conflict[Q](func() clause.Conflict {
		return conflict
	})
}

func (c Conflict[Q]) DoNothing() bob.Mod[Q] {
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

func (c Conflict[Q]) SetExcluded(cols ...string) Conflict[Q] {
	conflict := c()
	exprs := make([]any, 0, len(cols))
	for _, col := range cols {
		if col == "" {
			continue
		}
		exprs = append(exprs,
			expr.Join{Exprs: []bob.Expression{expr.Quote(col), expr.Raw("= EXCLUDED."), expr.Quote(col)}},
		)
	}
	conflict.Set.Set = append(conflict.Set.Set, exprs...)

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
