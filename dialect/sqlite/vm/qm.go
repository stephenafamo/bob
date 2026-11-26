package vm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func RowValue(clauses ...bob.Expression) bob.Mod[dialect.ValuesEditor] {
	return mods.Values[dialect.ValuesEditor](clauses)
}
