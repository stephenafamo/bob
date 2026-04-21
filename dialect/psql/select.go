package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type SelectQuery struct {
	derivedSelectQuery
}

func (q SelectQuery) With(queryMods ...bob.Mod[*dialect.SelectQuery]) SelectQuery {
	q.derivedSelectQuery = q.derivedSelectQuery.With(queryMods...)
	return q
}

func (q SelectQuery) Apply(queryMods ...bob.Mod[*dialect.SelectQuery]) SelectQuery {
	return q.With(queryMods...)
}

func Select(queryMods ...bob.Mod[*dialect.SelectQuery]) SelectQuery {
	state, ok := (immutableSelectState{}).withMods(queryMods...)
	if ok {
		return SelectQuery{
			derivedSelectQuery: derivedSelectQuery{
				state: state,
			},
		}
	}

	q := &dialect.SelectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return SelectQuery{
		derivedSelectQuery: asImmutable(bob.BaseQuery[*dialect.SelectQuery]{
			Expression: q,
			Dialect:    dialect.Dialect,
			QueryType:  bob.QueryTypeSelect,
		}),
	}
}
