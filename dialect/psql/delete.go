package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type DeleteQuery struct {
	bob.BaseQuery[*dialect.DeleteQuery]
}

func (q DeleteQuery) With(queryMods ...bob.Mod[*dialect.DeleteQuery]) DeleteQuery {
	if next, ok := q.Expression.Derive(queryMods...); ok {
		q.Expression = next
		return q
	}
	q.BaseQuery = q.BaseQuery.Apply(queryMods...)
	return q
}

func (q DeleteQuery) Apply(queryMods ...bob.Mod[*dialect.DeleteQuery]) DeleteQuery {
	return q.With(queryMods...)
}

func Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) DeleteQuery {
	q := &dialect.DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return DeleteQuery{
		BaseQuery: bob.BaseQuery[*dialect.DeleteQuery]{
			Expression: q,
			Dialect:    dialect.Dialect,
			QueryType:  bob.QueryTypeDelete,
		},
	}
}
