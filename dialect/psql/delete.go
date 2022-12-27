package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Delete(queryMods ...bob.Mod[*dialect.DeleteQuery]) bob.BaseQuery[*dialect.DeleteQuery] {
	q := &dialect.DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.DeleteQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
	}
}
