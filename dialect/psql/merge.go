package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type (
	MergeEditor = dialect.MergeEditor
	MergeQuery  = bob.QueryBuilder[MergeEditor]
	MergeMod    = bob.Mod[MergeEditor]
)

func Merge(mods ...MergeMod) MergeQuery {
	return dialect.NewMerge(nil, mods...)
}
