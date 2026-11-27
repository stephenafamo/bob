package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mssql/dialect"
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

func GroupByDistinct(distinct bool) bob.Mod[*dialect.SelectQuery] {
	return mods.GroupByDistinct[*dialect.SelectQuery](distinct)
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

// Top limits the number of rows returned
// SQL: SELECT TOP (count) ...
func Top(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.Top = &dialect.Top{Count: count}
	})
}

// TopPercent limits the result to a percentage of rows
// SQL: SELECT TOP (count) PERCENT ...
func TopPercent(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.Top = &dialect.Top{Count: count, Percent: true}
	})
}

// TopWithTies includes additional rows that tie with the last row (requires ORDER BY)
// SQL: SELECT TOP (count) WITH TIES ...
func TopWithTies(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.Top = &dialect.Top{Count: count, WithTies: true}
	})
}

func Offset(count any) bob.Mod[*dialect.SelectQuery] {
	return mods.Offset[*dialect.SelectQuery]{
		Count: count,
	}
}

// Fetch adds FETCH NEXT n ROWS ONLY for pagination.
func Fetch(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.Fetch.Count = count
	})
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

// To apply order to the result of a UNION, INTERSECT, or EXCEPT query
func OrderCombined(e any) dialect.OrderCombined {
	return dialect.OrderCombined(func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// To apply offset to the result of a UNION, INTERSECT, or EXCEPT query
func OffsetCombined(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.CombinedOffset.SetOffset(count)
	})
}

// To apply fetch to the result of a UNION, INTERSECT, or EXCEPT query.
func FetchCombined(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.CombinedFetch.Count = count
	})
}
