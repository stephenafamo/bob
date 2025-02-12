package mysql

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/dialect/mysql/dialect"
)

func Select(queryMods ...bob.Mod[*dialect.SelectQuery]) bob.BaseQuery[*dialect.SelectQuery] {
	q := &dialect.SelectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.SelectQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeSelect,
	}
}
