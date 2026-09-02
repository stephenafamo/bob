package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Listen(mods ...bob.Mod[*dialect.ListenQuery]) bob.BaseQuery[*dialect.ListenQuery] {
	q := &dialect.ListenQuery{}
	for _, mod := range mods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.ListenQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeListen,
	}
}
