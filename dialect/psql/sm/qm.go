package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

// With starts a CTE. The name and column list are quoted as SQL identifiers.
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

// From sets the query source. Pass a table name as a string (unquoted literal) or use
// psql.Quote(...) / a subquery Expression for quoted or qualified names.
func From(table any) dialect.FromChain[*dialect.SelectQuery] {
	return dialect.From[*dialect.SelectQuery](table)
}

// FromFunction returns an expression for sm.From when the source is one or more
// table functions (ROWS FROM when multiple).
func FromFunction(funcs ...*dialect.Function) bob.Expression {
	return dialect.TableFunctions(funcs...)
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

func CrossJoin(e any) dialect.JoinChain[*dialect.SelectQuery] {
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

func Grouping(groups ...any) clause.Grouping {
	return clause.Grouping{Groups: groups}
}

func Rollup(groups ...any) clause.GroupingSet {
	return clause.GroupingSet{Type: "ROLLUP", Groups: groups}
}

func Cube(groups ...any) clause.GroupingSet {
	return clause.GroupingSet{Type: "CUBE", Groups: groups}
}

func GroupingSets(groups ...any) clause.GroupingSet {
	return clause.GroupingSet{Type: "GROUPING SETS", Groups: groups}
}

// Window defines a named window for the query. The name is quoted as an SQL identifier.
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

func Fetch(count any, withTies bool) bob.Mod[*dialect.SelectQuery] {
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

// ForUpdate locks selected tables. Pass table names as psql.Quote(...) or another Expression.
func ForUpdate(tables ...any) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

// ForNoKeyUpdate locks selected tables without affecting key columns.
func ForNoKeyUpdate(tables ...any) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthNoKeyUpdate,
			Tables:   tables,
		}
	})
}

// ForShare acquires a shared lock on selected tables.
func ForShare(tables ...any) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}

// ForKeyShare acquires a key-share lock on selected tables.
func ForKeyShare(tables ...any) dialect.LockChain[*dialect.SelectQuery] {
	return dialect.LockChain[*dialect.SelectQuery](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthKeyShare,
			Tables:   tables,
		}
	})
}

// To apply order to the result of a UNION, INTERSECT, or EXCEPT query
func OrderCombined(e any) dialect.OrderCombined {
	return dialect.OrderCombined(func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// To apply limit to the result of a UNION, INTERSECT, or EXCEPT query
func LimitCombined(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.CombinedLimit.SetLimit(count)
	})
}

// To apply offset to the result of a UNION, INTERSECT, or EXCEPT query
func OffsetCombined(count any) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.CombinedOffset.SetOffset(count)
	})
}

// To apply fetch to the result of a UNION, INTERSECT, or EXCEPT query
func FetchCombined(count any, withTies bool) bob.Mod[*dialect.SelectQuery] {
	return bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
		q.CombinedFetch.Count = count
		q.CombinedFetch.WithTies = withTies
	})
}
