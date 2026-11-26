package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[dialect.SelectEditor] {
	return dialect.With[dialect.SelectEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.SelectEditor] {
	return mods.Recursive[dialect.SelectEditor](r)
}

func Distinct() bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetDistinct(true)
	})
}

func Columns(clauses ...any) bob.Mod[dialect.SelectEditor] {
	return mods.Select[dialect.SelectEditor](clauses)
}

func From(table any) dialect.FromChain[dialect.SelectEditor] {
	return dialect.From[dialect.SelectEditor](table)
}

func InnerJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
	return dialect.InnerJoin[dialect.SelectEditor](e)
}

func LeftJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
	return dialect.LeftJoin[dialect.SelectEditor](e)
}

func RightJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
	return dialect.RightJoin[dialect.SelectEditor](e)
}

func FullJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
	return dialect.FullJoin[dialect.SelectEditor](e)
}

func CrossJoin(e any) dialect.CrossJoinChain[dialect.SelectEditor] {
	return dialect.CrossJoin[dialect.SelectEditor](e)
}

func Where(e bob.Expression) mods.Where[dialect.SelectEditor] {
	return mods.Where[dialect.SelectEditor]{E: e}
}

func Having(e any) bob.Mod[dialect.SelectEditor] {
	return mods.Having[dialect.SelectEditor]{e}
}

func GroupBy(e any) bob.Mod[dialect.SelectEditor] {
	return mods.GroupBy[dialect.SelectEditor]{
		E: e,
	}
}

// Window defines a named window. The name is quoted as an SQL identifier.
func Window(name string, winMods ...bob.Mod[*clause.Window]) bob.Mod[dialect.SelectEditor] {
	w := clause.Window{}
	for _, mod := range winMods {
		mod.Edit(&w)
	}

	return mods.NamedWindow[dialect.SelectEditor](clause.NamedWindow{
		Name:       name,
		Definition: w,
	})
}

func OrderBy(e any) dialect.OrderBy[dialect.SelectEditor] {
	return dialect.OrderBy[dialect.SelectEditor](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// Sqlite can use an clauseession for the limit
func Limit(count any) bob.Mod[dialect.SelectEditor] {
	return mods.Limit[dialect.SelectEditor]{
		Count: count,
	}
}

// Sqlite can use an clauseession for the offset
func Offset(count any) bob.Mod[dialect.SelectEditor] {
	return mods.Offset[dialect.SelectEditor]{
		Count: count,
	}
}

func Union(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func UnionAll(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func Intersect(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func Except(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}
