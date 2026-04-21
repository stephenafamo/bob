package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type DeleteQuery struct {
	derivedDeleteQuery
	materialized *bob.BaseQuery[*dialect.DeleteQuery]
}

func (q DeleteQuery) With(queryMods ...bob.Mod[*dialect.DeleteQuery]) DeleteQuery {
	q.derivedDeleteQuery = q.derivedDeleteQuery.With(queryMods...)
	q.materialized = nil
	return q
}

func (q DeleteQuery) Apply(queryMods ...bob.Mod[*dialect.DeleteQuery]) DeleteQuery {
	return q.With(queryMods...)
}

func (q DeleteQuery) baseQuery() bob.BaseQuery[*dialect.DeleteQuery] {
	if q.materialized != nil {
		return *q.materialized
	}
	return q.derivedDeleteQuery.mutableBase()
}

func Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) DeleteQuery {
	q := &dialect.DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	base := bob.BaseQuery[*dialect.DeleteQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeDelete,
	}

	return DeleteQuery{
		derivedDeleteQuery: asImmutableDelete(base),
		materialized:       &base,
	}
}
