package vm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
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

func Limit(count any) bob.Mod[dialect.ValuesEditor] {
	return mods.Limit[dialect.ValuesEditor]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[dialect.ValuesEditor] {
	return mods.Offset[dialect.ValuesEditor]{
		Count: count,
	}
}

func Fetch(count any) bob.Mod[dialect.ValuesEditor] {
	return mods.Fetch[dialect.ValuesEditor]{
		Count: count,
	}
}
