package psql

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/dialect/psql/dialect"
)

func Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) bob.BaseQuery[*dialect.DeleteQuery] {
	q := &dialect.DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.DeleteQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeDelete,
	}
}
