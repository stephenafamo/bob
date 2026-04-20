package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type SelectQuery struct {
	bob.BaseQuery[*dialect.SelectQuery]
}

func (q SelectQuery) With(queryMods ...bob.Mod[*dialect.SelectQuery]) ImmutableSelectQuery {
	return asImmutable(q.BaseQuery).With(queryMods...)
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
