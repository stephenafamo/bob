package vm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mssql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func RowValue(clauses ...bob.Expression) bob.Mod[*dialect.ValuesQuery] {
	return mods.Values[*dialect.ValuesQuery](clauses)
}

func OrderBy(e any) dialect.OrderBy[*dialect.ValuesQuery] {
	return dialect.OrderBy[*dialect.ValuesQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Offset(count any) bob.Mod[*dialect.ValuesQuery] {
	return mods.Offset[*dialect.ValuesQuery]{
		Count: count,
	}
}

func Fetch(count any) bob.Mod[*dialect.ValuesQuery] {
	return bob.ModFunc[*dialect.ValuesQuery](func(q *dialect.ValuesQuery) {
		q.Fetch.Count = count
	})
}
