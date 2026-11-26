package mysql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

type (
	DeleteEditor = dialect.DeleteEditor
	DeleteQuery  = bob.QueryBuilder[DeleteEditor]
	DeleteMod    = bob.Mod[DeleteEditor]
)

func Delete(mods ...DeleteMod) DeleteQuery {
	return dialect.NewDelete(nil, mods...)
}
