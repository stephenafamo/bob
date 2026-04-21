package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type InsertQuery struct {
	derivedInsertQuery
}

func (q InsertQuery) With(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	q.derivedInsertQuery = q.derivedInsertQuery.With(queryMods...)
	return q
}

func (q InsertQuery) Apply(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	return q.With(queryMods...)
}

func Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	state, ok := (immutableInsertState{}).withMods(queryMods...)
	if ok {
		return InsertQuery{
			derivedInsertQuery: derivedInsertQuery{
				state: state,
			},
		}
	}

	q := &dialect.InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return InsertQuery{
		derivedInsertQuery: asImmutableInsert(bob.BaseQuery[*dialect.InsertQuery]{
			Expression: q,
			Dialect:    dialect.Dialect,
			QueryType:  bob.QueryTypeInsert,
		}),
	}
}
