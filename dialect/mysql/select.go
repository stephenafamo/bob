package mysql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

type (
	SelectEditor = dialect.SelectEditor
	SelectQuery  = bob.QueryBuilder[SelectEditor]
	SelectMod    = bob.Mod[SelectEditor]
)

func Select(mods ...SelectMod) SelectQuery {
	return dialect.NewSelect(nil, mods...)
}
