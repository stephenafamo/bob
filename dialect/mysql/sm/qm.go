package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[dialect.SelectEditor] {
	return dialect.With[dialect.SelectEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.SelectEditor] {
	return mods.Recursive[dialect.SelectEditor](r)
}

func Distinct(on ...any) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.AppendModifier("DISTINCT")
	})
}

func HighPriority() bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.AppendModifier("HIGH_PRIORITY")
	})
}

func Straight() bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.AppendModifier("STRAIGHT_JOIN")
	})
}

func SmallResult() bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.AppendModifier("SQL_SMALL_RESULT")
	})
}

func BigResult() bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.AppendModifier("SQL_BIG_RESULT")
	})
}

func BufferResult() bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.AppendModifier("SQL_BUFFER_RESULT")
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

func CrossJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
	return dialect.CrossJoin[dialect.SelectEditor](e)
}

func StraightJoin(e any) dialect.JoinChain[dialect.SelectEditor] {
	return dialect.StraightJoin[dialect.SelectEditor](e)
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

func WithRollup(distinct bool) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetGroupWith("ROLLUP")
	})
}

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

func Limit(count int64) bob.Mod[dialect.SelectEditor] {
	return mods.Limit[dialect.SelectEditor]{
		Count: count,
	}
}

func Offset(count int64) bob.Mod[dialect.SelectEditor] {
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

// ForUpdate locks selected tables. Pass table names as mysql.Quote(...) or another Expression.
func ForUpdate(tables ...any) dialect.LockChain[dialect.SelectEditor] {
	return dialect.LockChain[dialect.SelectEditor](func() clause.Lock {
		return clause.Lock{
			Strength: clause.LockStrengthUpdate,
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

// To apply order to the result of a UNION, INTERSECT, or EXCEPT query
func OrderCombined(e any) dialect.OrderCombined {
	return dialect.OrderCombined(func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// To apply limit to the result of a UNION, INTERSECT, or EXCEPT query
func LimitCombined(count int64) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetCombinedLimit(count)
	})
}

// To apply offset to the result of a UNION, INTERSECT, or EXCEPT query
func OffsetCombined(count int64) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetCombinedOffset(count)
	})
}

// No need for the leading @
func Into(var1 string, vars ...string) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetInto(into{
			vars: append([]string{var1}, vars...),
		})
	})
}

func IntoDumpfile(filename string) bob.Mod[dialect.SelectEditor] {
	return bob.ModFunc[dialect.SelectEditor](func(q dialect.SelectEditor) {
		q.SetInto(into{
			dumpfile: filename,
		})
	})
}

func IntoOutfile(filename string) *intoChain {
	return &intoChain{
		into: into{
			outfile: filename,
		},
	}
}
