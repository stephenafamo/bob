package mods

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

type Conflict[Q interface{ SetConflict(bob.Expression) }] func() clause.ConflictClause

// ConflictColumns creates an ON CONFLICT clause initialized with target columns.
// Additional target details can be provided by passing richer target item values
// to dialect-level OnConflict(...) helpers.
func ConflictColumns[Q interface{ SetConflict(bob.Expression) }](columns ...any) Conflict[Q] {
	return Conflict[Q](func() clause.ConflictClause {
		return clause.ConflictClause{
			Target: clause.ConflictTarget{
				Columns: columns,
			},
		}
	})
}

// ConflictOnConstraint creates an ON CONFLICT clause initialized with
// ON CONSTRAINT <constraint> target selection.
func ConflictOnConstraint[Q interface{ SetConflict(bob.Expression) }](constraint string) Conflict[Q] {
	return Conflict[Q](func() clause.ConflictClause {
		return clause.ConflictClause{
			Target: clause.ConflictTarget{
				Constraint: constraint,
			},
		}
	})
}

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
