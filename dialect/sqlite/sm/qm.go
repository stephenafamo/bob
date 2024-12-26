package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.SelectQuery] {
	return dialect.With[*dialect.SelectQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.SelectQuery] {
	return mods.Recursive[*dialect.SelectQuery](r)
}

func Distinct() bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.Distinct = true
	})
}

func Columns(clauses ...any) bob.Mod[*dialect.SelectQuery] {
	return mods.Select[*dialect.SelectQuery](clauses)
}

func From(table any) dialect.FromChain[*dialect.SelectQuery] {
	return dialect.From[*dialect.SelectQuery](table)
}

func InnerJoin(e any) dialect.JoinChain[*dialect.SelectQuery] {
	return dialect.InnerJoin[*dialect.SelectQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*dialect.SelectQuery] {
	return dialect.LeftJoin[*dialect.SelectQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*dialect.SelectQuery] {
	return dialect.RightJoin[*dialect.SelectQuery](e)
}

func FullJoin(e any) dialect.JoinChain[*dialect.SelectQuery] {
	return dialect.FullJoin[*dialect.SelectQuery](e)
}

func CrossJoin(e any) dialect.CrossJoinChain[*dialect.SelectQuery] {
	return dialect.CrossJoin[*dialect.SelectQuery](e)
}

func Where(e bob.Expression) mods.Where[*dialect.SelectQuery] {
	return mods.Where[*dialect.SelectQuery]{E: e}
}

func Having(e any) bob.Mod[*dialect.SelectQuery] {
	return mods.Having[*dialect.SelectQuery]{e}
}

func GroupBy(e any) bob.Mod[*dialect.SelectQuery] {
	return mods.GroupBy[*dialect.SelectQuery]{
		E: e,
	}
}

func Window(name string, winMods ...bob.Mod[*clause.Window]) bob.Mod[*dialect.SelectQuery] {
	w := clause.Window{}
	for _, mod := range winMods {
		mod.Apply(&w)
	}

	return mods.NamedWindow[*dialect.SelectQuery](clause.NamedWindow{
		Name:       name,
		Definition: w,
	})
}

func OrderBy(e any) dialect.OrderBy[*dialect.SelectQuery] {
	return dialect.OrderBy[*dialect.SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// Sqlite can use an clauseession for the limit
func Limit(count any) bob.Mod[*dialect.SelectQuery] {
	return mods.Limit[*dialect.SelectQuery]{
		Count: count,
	}
}

// Sqlite can use an clauseession for the offset
func Offset(count any) bob.Mod[*dialect.SelectQuery] {
	return mods.Offset[*dialect.SelectQuery]{
		Count: count,
	}
}

func Union(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func UnionAll(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func Intersect(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func Except(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}
