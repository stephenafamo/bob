package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type DeleteQuery struct {
	bob.BaseQuery[*dialect.DeleteQuery]
}

func (q DeleteQuery) With(queryMods ...bob.Mod[*dialect.DeleteQuery]) derivedDeleteQuery {
	return asImmutableDelete(q.BaseQuery).With(queryMods...)
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
