package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*mysql.SelectQuery] {
	return dialect.With[*mysql.SelectQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*mysql.SelectQuery] {
	return mods.Recursive[*mysql.SelectQuery](r)
}

func Distinct(on ...any) bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.AppendModifier("DISTINCT")
	})
}

func HighPriority() bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.AppendModifier("HIGH_PRIORITY")
	})
}

func Straight() bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.AppendModifier("STRAIGHT_JOIN")
	})
}

func SmallResult() bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.AppendModifier("SQL_SMALL_RESULT")
	})
}

func BigResult() bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.AppendModifier("SQL_BIG_RESULT")
	})
}

func BufferResult() bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.AppendModifier("SQL_BUFFER_RESULT")
	})
}

func Columns(clauses ...any) bob.Mod[*mysql.SelectQuery] {
	return mods.Select[*mysql.SelectQuery](clauses)
}

func From(table any) bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.SetTable(table)
	})
}

func As(alias string, columns ...string) bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.SetTableAlias(alias, columns...)
	})
}

func Lateral() bob.Mod[*mysql.SelectQuery] {
	return dialect.Lateral[*mysql.SelectQuery]()
}

func UseIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.SelectQuery] {
	return dialect.UseIndex[*mysql.SelectQuery](first, others...)
}

func IgnoreIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.SelectQuery] {
	return dialect.UseIndex[*mysql.SelectQuery](first, others...)
}

func ForceIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.SelectQuery] {
	return dialect.UseIndex[*mysql.SelectQuery](first, others...)
}

func Partition(partitions ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Partition[*mysql.InsertQuery](partitions...)
}

func InnerJoin(e bob.Expression) dialect.JoinChain[*mysql.DeleteQuery] {
	return dialect.InnerJoin[*mysql.DeleteQuery](e)
}

func LeftJoin(e bob.Expression) dialect.JoinChain[*mysql.DeleteQuery] {
	return dialect.LeftJoin[*mysql.DeleteQuery](e)
}

func RightJoin(e bob.Expression) dialect.JoinChain[*mysql.DeleteQuery] {
	return dialect.RightJoin[*mysql.DeleteQuery](e)
}

func CrossJoin(e bob.Expression) bob.Mod[*mysql.DeleteQuery] {
	return dialect.CrossJoin[*mysql.DeleteQuery](e)
}

func StraightJoin(e bob.Expression) bob.Mod[*mysql.DeleteQuery] {
	return dialect.StraightJoin[*mysql.DeleteQuery](e)
}

func Where(e bob.Expression) bob.Mod[*mysql.SelectQuery] {
	return mods.Where[*mysql.SelectQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*mysql.SelectQuery] {
	return mods.Where[*mysql.SelectQuery]{mysql.Raw(clause, args...)}
}

func Having(e bob.Expression) bob.Mod[*mysql.SelectQuery] {
	return mods.Having[*mysql.SelectQuery]{e}
}

func HavingClause(clause string, args ...any) bob.Mod[*mysql.SelectQuery] {
	return mods.Having[*mysql.SelectQuery]{mysql.Raw(clause, args...)}
}

func GroupBy(e any) bob.Mod[*mysql.SelectQuery] {
	return mods.GroupBy[*mysql.SelectQuery]{
		E: e,
	}
}

func WithRollup(distinct bool) bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.SetGroupWith("ROLLUP")
	})
}

func Window(name string) dialect.WindowMod[*mysql.SelectQuery] {
	m := dialect.WindowMod[*mysql.SelectQuery]{
		Name: name,
	}

	m.WindowChain = &dialect.WindowChain[*dialect.WindowMod[*mysql.SelectQuery]]{
		Wrap: &m,
	}
	return m
}

func OrderBy(e any) dialect.OrderBy[*mysql.SelectQuery] {
	return dialect.OrderBy[*mysql.SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*mysql.SelectQuery] {
	return mods.Limit[*mysql.SelectQuery]{
		Count: count,
	}
}

func Offset(count int64) bob.Mod[*mysql.SelectQuery] {
	return mods.Offset[*mysql.SelectQuery]{
		Count: count,
	}
}

func Union(q bob.Query) bob.Mod[*mysql.SelectQuery] {
	return mods.Combine[*mysql.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func UnionAll(q bob.Query) bob.Mod[*mysql.SelectQuery] {
	return mods.Combine[*mysql.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func ForUpdate(tables ...string) dialect.LockChain[*mysql.SelectQuery] {
	return dialect.LockChain[*mysql.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

func ForShare(tables ...string) dialect.LockChain[*mysql.SelectQuery] {
	return dialect.LockChain[*mysql.SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}

// No need for the leading @
func Into(var1 string, vars ...string) bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
		q.SetInto(into{
			vars: append([]string{var1}, vars...),
		})
	})
}

func IntoDumpfile(filename string) bob.Mod[*mysql.SelectQuery] {
	return mods.QueryModFunc[*mysql.SelectQuery](func(q *mysql.SelectQuery) {
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
