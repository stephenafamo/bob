package sqlite

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/dialect/sqlite/dialect"
)

func Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) bob.BaseQuery[*dialect.InsertQuery] {
	q := &dialect.InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.InsertQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeInsert,
	}
}
