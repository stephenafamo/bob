package vm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func ValueRow(clauses ...bob.Expression) bob.Mod[*dialect.ValuesQuery] {
	return mods.Values[*dialect.ValuesQuery](clauses)
}

func OrderBy(e any) dialect.OrderBy[*dialect.ValuesQuery] {
	return dialect.OrderBy[*dialect.ValuesQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count any) bob.Mod[*dialect.ValuesQuery] {
	return mods.Limit[*dialect.ValuesQuery]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[*dialect.ValuesQuery] {
	return mods.Offset[*dialect.ValuesQuery]{
		Count: count,
	}
}

func Fetch(count any) bob.Mod[*dialect.ValuesQuery] {
	return mods.Fetch[*dialect.ValuesQuery]{
		Count: count,
	}
}
