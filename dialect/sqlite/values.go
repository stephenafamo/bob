package sqlite

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
)

type (
	ValuesEditor = dialect.ValuesEditor
	ValuesQuery  = bob.QueryBuilder[ValuesEditor]
	ValuesMod    = bob.Mod[ValuesEditor]
)

func Values(mods ...ValuesMod) ValuesQuery {
	return dialect.NewValues(nil, mods...)
}
