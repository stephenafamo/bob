package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Notify(mods ...bob.Mod[*dialect.NotifyQuery]) bob.BaseQuery[*dialect.NotifyQuery] {
	q := &dialect.NotifyQuery{}
	for _, mod := range mods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.NotifyQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeNotify,
	}
}
