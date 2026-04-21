package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type UpdateQuery struct {
	derivedUpdateQuery
	materialized *bob.BaseQuery[*dialect.UpdateQuery]
}

func (q UpdateQuery) With(queryMods ...bob.Mod[*dialect.UpdateQuery]) UpdateQuery {
	q.derivedUpdateQuery = q.derivedUpdateQuery.With(queryMods...)
	q.materialized = nil
	return q
}

func (q UpdateQuery) Apply(queryMods ...bob.Mod[*dialect.UpdateQuery]) UpdateQuery {
	return q.With(queryMods...)
}

func (q UpdateQuery) baseQuery() bob.BaseQuery[*dialect.UpdateQuery] {
	if q.materialized != nil {
		return *q.materialized
	}
	return q.derivedUpdateQuery.mutableBase()
}

func Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) UpdateQuery {
	q := &dialect.UpdateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	base := bob.BaseQuery[*dialect.UpdateQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeUpdate,
	}

	return UpdateQuery{
		derivedUpdateQuery: asImmutableUpdate(base),
		materialized:       &base,
	}
}
