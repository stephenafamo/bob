package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*psql.SelectQuery] {
	return dialect.With[*psql.SelectQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*psql.SelectQuery] {
	return mods.Recursive[*psql.SelectQuery](r)
}

func Distinct(on ...any) bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		q.Select.Modifiers = []any{
			dialect.Distinct{On: on},
		}
	})
}

func Columns(clauses ...any) bob.Mod[*psql.SelectQuery] {
	return mods.Select[*psql.SelectQuery](clauses)
}

func From(table any) bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		q.SetTable(table)
	})
}

func FromFunction(funcs ...*dialect.Function) bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		if len(funcs) == 0 {
			return
		}
		if len(funcs) == 1 {
			q.SetTable(funcs[0])
			return
		}

		q.SetTable(dialect.Functions(funcs))
	})
}

func As(alias string, columns ...string) bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		q.SetTableAlias(alias, columns...)
	})
}

func Only() bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		q.SetOnly(true)
	})
}

func Lateral() bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		q.SetLateral(true)
	})
}

func WithOrdinality() bob.Mod[*psql.SelectQuery] {
	return mods.QueryModFunc[*psql.SelectQuery](func(q *psql.SelectQuery) {
		q.SetWithOrdinality(true)
	})
}

func InnerJoin(e any) dialect.JoinChain[*psql.SelectQuery] {
	return dialect.InnerJoin[*psql.SelectQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*psql.SelectQuery] {
	return dialect.LeftJoin[*psql.SelectQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*psql.SelectQuery] {
	return dialect.RightJoin[*psql.SelectQuery](e)
}

func FullJoin(e any) dialect.JoinChain[*psql.SelectQuery] {
	return dialect.FullJoin[*psql.SelectQuery](e)
}

func CrossJoin(e any) bob.Mod[*psql.SelectQuery] {
	return dialect.CrossJoin[*psql.SelectQuery](e)
}

func Where(e bob.Expression) bob.Mod[*psql.SelectQuery] {
	return mods.Where[*psql.SelectQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*psql.SelectQuery] {
	return mods.Where[*psql.SelectQuery]{psql.Raw(clause, args...)}
}

func Having(e bob.Expression) bob.Mod[*psql.SelectQuery] {
	return mods.Having[*psql.SelectQuery]{e}
}

func HavingClause(clause string, args ...any) bob.Mod[*psql.SelectQuery] {
	return mods.Having[*psql.SelectQuery]{psql.Raw(clause, args...)}
}

func GroupBy(e any) bob.Mod[*psql.SelectQuery] {
	return mods.GroupBy[*psql.SelectQuery]{
		E: e,
	}
}

func GroupByDistinct(distinct bool) bob.Mod[*psql.SelectQuery] {
	return mods.GroupByDistinct[*psql.SelectQuery](distinct)
}

func Window(name string) dialect.WindowMod[*psql.SelectQuery] {
	m := dialect.WindowMod[*psql.SelectQuery]{
		Name: name,
	}

	m.WindowChain = &dialect.WindowChain[*dialect.WindowMod[*psql.SelectQuery]]{
		Wrap: &m,
	}
	return m
}

func OrderBy(e any) dialect.OrderBy[*psql.SelectQuery] {
	return dialect.OrderBy[*psql.SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*psql.SelectQuery] {
	return mods.Limit[*psql.SelectQuery]{
		Count: count,
	}
}

func Offset(count int64) bob.Mod[*psql.SelectQuery] {
	return mods.Offset[*psql.SelectQuery]{
		Count: count,
	}
}

func Fetch(count int64, withTies bool) bob.Mod[*psql.SelectQuery] {
	return mods.Fetch[*psql.SelectQuery]{
		Count:    &count,
		WithTies: withTies,
	}
}

func Union(q bob.Query) bob.Mod[*psql.SelectQuery] {
	return mods.Combine[*psql.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func UnionAll(q bob.Query) bob.Mod[*psql.SelectQuery] {
	return mods.Combine[*psql.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func Intersect(q bob.Query) bob.Mod[*psql.SelectQuery] {
	return mods.Combine[*psql.SelectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func IntersectAll(q bob.Query) bob.Mod[*psql.SelectQuery] {
	return mods.Combine[*psql.SelectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      true,
	}
}

func Except(q bob.Query) bob.Mod[*psql.SelectQuery] {
	return mods.Combine[*psql.SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}

func ExceptAll(q bob.Query) bob.Mod[*psql.SelectQuery] {
	return mods.Combine[*psql.SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      true,
	}
}

func ForUpdate(tables ...string) dialect.LockChain[*psql.SelectQuery] {
	return dialect.LockChain[*psql.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

func ForNoKeyUpdate(tables ...string) dialect.LockChain[*psql.SelectQuery] {
	return dialect.LockChain[*psql.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthNoKeyUpdate,
			Tables:   tables,
		}
	})
}

func ForShare(tables ...string) dialect.LockChain[*psql.SelectQuery] {
	return dialect.LockChain[*psql.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}

func ForKeyShare(tables ...string) dialect.LockChain[*psql.SelectQuery] {
	return dialect.LockChain[*psql.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthKeyShare,
			Tables:   tables,
		}
	})
}
