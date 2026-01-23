package mysql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func Values(queryMods ...bob.Mod[*dialect.ValuesQuery]) bob.BaseQuery[*dialect.ValuesQuery] {
	q := &dialect.ValuesQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.ValuesQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeValues,
	}
}
