package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type UpdateQuery struct {
	bob.BaseQuery[*dialect.UpdateQuery]
}

func (q UpdateQuery) With(queryMods ...bob.Mod[*dialect.UpdateQuery]) UpdateQuery {
	if next, ok := deriveUpdate(q.Expression, queryMods...); ok {
		q.Expression = next
		return q
	}
	q.BaseQuery = q.BaseQuery.Apply(queryMods...)
	return q
}

func (q UpdateQuery) Apply(queryMods ...bob.Mod[*dialect.UpdateQuery]) UpdateQuery {
	return q.With(queryMods...)
}

func Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) UpdateQuery {
	q := &dialect.UpdateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return UpdateQuery{
		BaseQuery: bob.BaseQuery[*dialect.UpdateQuery]{
			Expression: q,
			Dialect:    dialect.Dialect,
			QueryType:  bob.QueryTypeUpdate,
		},
	}
}
