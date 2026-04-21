package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type InsertQuery struct {
	bob.BaseQuery[*dialect.InsertQuery]
}

func (q InsertQuery) With(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	if next, ok := q.Expression.Derive(queryMods...); ok {
		q.Expression = next
		return q
	}
	q.BaseQuery = q.BaseQuery.Apply(queryMods...)
	return q
}

func (q InsertQuery) Apply(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	return q.With(queryMods...)
}

func Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) InsertQuery {
	q := &dialect.InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return InsertQuery{
		BaseQuery: bob.BaseQuery[*dialect.InsertQuery]{
			Expression: q,
			Dialect:    dialect.Dialect,
			QueryType:  bob.QueryTypeInsert,
		},
	}
}
