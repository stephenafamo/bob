package vm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func RowValue(clauses ...bob.Expression) bob.Mod[dialect.ValuesEditor] {
	return mods.Values[dialect.ValuesEditor](clauses)
}

func OrderBy(e any) dialect.OrderBy[dialect.ValuesEditor] {
	return dialect.OrderBy[dialect.ValuesEditor](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[dialect.ValuesEditor] {
	return mods.Limit[dialect.ValuesEditor]{
		Count: count,
	}
}
