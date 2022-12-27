package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Insert(queryMods ...bob.Mod[*dialect.InsertQuery]) bob.BaseQuery[*dialect.InsertQuery] {
	q := &dialect.InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.InsertQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
	}
}
