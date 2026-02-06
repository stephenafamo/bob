package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Merge(queryMods ...bob.Mod[*dialect.MergeQuery]) bob.BaseQuery[*dialect.MergeQuery] {
	q := &dialect.MergeQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*dialect.MergeQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
		QueryType:  bob.QueryTypeMerge,
	}
}
