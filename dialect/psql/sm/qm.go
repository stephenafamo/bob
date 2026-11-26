package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

// With starts a CTE. The name and column list are quoted as SQL identifiers.
func With(name string, columns ...string) dialect.CTEChain[dialect.SelectEditor] {
	return dialect.With[dialect.SelectEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.SelectEditor] {
	return mods.Recursive[dialect.SelectEditor](r)
}

func Distinct(on ...any) bob.Mod[dialect.SelectEditor] {
	if on == nil {
		on = []any{} // nil means no distinct
	}

	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetDistinctOn(on...)
	})
}

func Columns(clauses ...any) bob.Mod[dialect.SelectEditor] {
	return mods.Select[dialect.SelectEditor](clauses)
}

// From sets the query source. Pass a table name as a string (unquoted literal) or use
// psql.Quote(...) / a subquery Expression for quoted or qualified names.
// Optional joins are attached to this from_item.
func From(table any, joins ...dialect.JoinChain[dialect.SelectEditor]) dialect.FromChain[dialect.SelectEditor] {
	return dialect.From(table, joins...)
}

// FromFunction returns an expression for sm.From when the source is one or more
// table functions (ROWS FROM when multiple).
func FromFunction(funcs ...*dialect.Function) bob.Expression {
	return dialect.TableFunctions(funcs...)
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

func CrossJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
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

func GroupByDistinct(distinct bool) bob.Mod[dialect.SelectEditor] {
	return mods.GroupByDistinct[dialect.SelectEditor](distinct)
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

func Limit(count any) bob.Mod[dialect.SelectEditor] {
	return mods.Limit[dialect.SelectEditor]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[dialect.SelectEditor] {
	return mods.Offset[dialect.SelectEditor]{
		Count: count,
	}
}

func Fetch(count any, withTies bool) bob.Mod[dialect.SelectEditor] {
	return mods.Fetch[dialect.SelectEditor]{
		Count:    &count,
		WithTies: withTies,
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

func IntersectAll(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      true,
	}
}

func Except(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}

func ExceptAll(q bob.Query) bob.Mod[dialect.SelectEditor] {
	return mods.Combine[dialect.SelectEditor]{
		Strategy: clause.Except,
		Query:    q,
		All:      true,
	}
}

// ForUpdate locks selected tables. Pass table names as psql.Quote(...) or another Expression.
func ForUpdate(tables ...any) dialect.LockChain[dialect.SelectEditor] {
	return dialect.LockChain[dialect.SelectEditor](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

// ForNoKeyUpdate locks selected tables without affecting key columns.
func ForNoKeyUpdate(tables ...any) dialect.LockChain[dialect.SelectEditor] {
	return dialect.LockChain[dialect.SelectEditor](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthNoKeyUpdate,
			Tables:   tables,
		}
	})
}

// ForShare acquires a shared lock on selected tables.
func ForShare(tables ...any) dialect.LockChain[dialect.SelectEditor] {
	return dialect.LockChain[dialect.SelectEditor](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}

// ForKeyShare acquires a key-share lock on selected tables.
func ForKeyShare(tables ...any) dialect.LockChain[dialect.SelectEditor] {
	return dialect.LockChain[dialect.SelectEditor](func() clause.Lock {
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
func LimitCombined(count any) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetCombinedLimit(count)
	})
}

// To apply offset to the result of a UNION, INTERSECT, or EXCEPT query
func OffsetCombined(count any) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetCombinedOffset(count)
	})
}

// To apply fetch to the result of a UNION, INTERSECT, or EXCEPT query
func FetchCombined(count any, withTies bool) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetCombinedFetch(clause.Fetch{Count: count, WithTies: withTies})
	})
}
