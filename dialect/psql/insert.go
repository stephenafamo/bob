package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type (
	InsertEditor = dialect.InsertEditor
	InsertQuery  = bob.QueryBuilder[InsertEditor]
	InsertMod    = bob.Mod[InsertEditor]
)

func Insert(mods ...InsertMod) InsertQuery {
	return dialect.NewInsert(nil, mods...)
}
