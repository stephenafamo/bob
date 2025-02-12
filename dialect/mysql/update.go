package mysql

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/dialect/mysql/dialect"
)

func Update(queryMods ...bob.Mod[*dialect.UpdateQuery]) bob.BaseQuery[*dialect.UpdateQuery] {
	q := &dialect.UpdateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.UpdateQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeUpdate,
	}
}
