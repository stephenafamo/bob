package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.SelectQuery] {
	return dialect.With[*dialect.SelectQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.SelectQuery] {
	return mods.Recursive[*dialect.SelectQuery](r)
}

func Distinct(on ...any) bob.Mod[*dialect.SelectQuery] {
	if on == nil {
		on = []any{} // nil means no distinct
	}

	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.Distinct.On = on
	})
}

func Columns(clauses ...any) bob.Mod[*dialect.SelectQuery] {
	return mods.Select[*dialect.SelectQuery](clauses)
}

func From(table any) dialect.FromChain[*dialect.SelectQuery] {
	return dialect.From[*dialect.SelectQuery](table)
}

func FromFunction(funcs ...*dialect.Function) dialect.FromChain[*dialect.SelectQuery] {
	var table any

	if len(funcs) == 1 {
		table = funcs[0]
	}

	if len(funcs) > 1 {
		table = dialect.Functions(funcs)
	}

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

func GroupByDistinct(distinct bool) bob.Mod[*dialect.SelectQuery] {
	return mods.GroupByDistinct[*dialect.SelectQuery](distinct)
}

func Window(name string) dialect.WindowsMod[*dialect.SelectQuery] {
	m := dialect.WindowsMod[*dialect.SelectQuery]{
		Name: name,
	}

	m.WindowChain = &dialect.WindowChain[*dialect.WindowsMod[*dialect.SelectQuery]]{
		Wrap: &m,
	}
	return m
}

func OrderBy(e any) dialect.OrderBy[*dialect.SelectQuery] {
	return dialect.OrderBy[*dialect.SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count any) bob.Mod[*dialect.SelectQuery] {
	return mods.Limit[*dialect.SelectQuery]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[*dialect.SelectQuery] {
	return mods.Offset[*dialect.SelectQuery]{
		Count: count,
	}
}

func Fetch(count int64, withTies bool) bob.Mod[*dialect.SelectQuery] {
	return mods.Fetch[*dialect.SelectQuery]{
		Count:    &count,
		WithTies: withTies,
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

func IntersectAll(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      true,
	}
}

func Except(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}

func ExceptAll(q bob.Query) bob.Mod[*dialect.SelectQuery] {
	return mods.Combine[*dialect.SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      true,
	}
}

func ForUpdate(tables ...string) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

func ForNoKeyUpdate(tables ...string) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthNoKeyUpdate,
			Tables:   tables,
		}
	})
}

func ForShare(tables ...string) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}

func ForKeyShare(tables ...string) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthKeyShare,
			Tables:   tables,
		}
	})
}
