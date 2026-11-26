package mysql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

type (
	UpdateEditor = dialect.UpdateEditor
	UpdateQuery  = bob.QueryBuilder[UpdateEditor]
	UpdateMod    = bob.Mod[UpdateEditor]
)

func Update(mods ...UpdateMod) UpdateQuery {
	return dialect.NewUpdate(nil, mods...)
}
