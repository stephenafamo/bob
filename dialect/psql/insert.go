package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type InsertQuery struct {
	derivedInsertQuery
	materialized *bob.BaseQuery[*dialect.InsertQuery]
}

func (q InsertQuery) With(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	q.derivedInsertQuery = q.derivedInsertQuery.With(queryMods...)
	q.materialized = nil
	return q
}

func (q InsertQuery) Apply(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	return q.With(queryMods...)
}

func (q InsertQuery) baseQuery() bob.BaseQuery[*dialect.InsertQuery] {
	if q.materialized != nil {
		return *q.materialized
	}
	return q.derivedInsertQuery.mutableBase()
}

func Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	q := &dialect.InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	base := bob.BaseQuery[*dialect.InsertQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeInsert,
	}

	return InsertQuery{
		derivedInsertQuery: asImmutableInsert(base),
		materialized:       &base,
	}
}
