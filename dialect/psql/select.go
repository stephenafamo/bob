package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type SelectQuery struct {
	bob.BaseQuery[*dialect.SelectQuery]
}

func (q SelectQuery) Apply(queryMods ...bob.Mod[*dialect.SelectQuery]) SelectQuery {
	if next, ok := q.Expression.Derive(queryMods...); ok {
		q.Expression = next
		return q
	}
	q.BaseQuery = q.BaseQuery.Apply(queryMods...)
	return q
}

func (q SelectQuery) AsCount() SelectQuery {
	next := q.Clone()
	next.Expression.SetSelect("count(1)")
	next.Expression.SetPreloadSelect()
	next.Expression.SetMapperMods()
	next.Expression.SetLoaders()
	next.Expression.SetLimit(1)
	next.Expression.ClearOrderBy()
	next.Expression.SetGroups()
	next.Expression.SetOffset(0)
	return SelectQuery{BaseQuery: next}
}

func Select(queryMods ...bob.Mod[*dialect.SelectQuery]) SelectQuery {
	q := &dialect.SelectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return SelectQuery{
		BaseQuery: bob.BaseQuery[*dialect.SelectQuery]{
			Expression: q,
			Dialect:    dialect.Dialect,
			QueryType:  bob.QueryTypeSelect,
		},
	}
}
