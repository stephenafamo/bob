package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*mysql.DeleteQuery] {
	return dialect.With[*mysql.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*mysql.DeleteQuery] {
	return mods.Recursive[*mysql.DeleteQuery](r)
}

func LowPriority() bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(i *mysql.DeleteQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Quick() bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(i *mysql.DeleteQuery) {
		i.AppendModifier("QUICK")
	})
}

func Ignore() bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(i *mysql.DeleteQuery) {
		i.AppendModifier("IGNORE")
	})
}

func UsingAs(alias string, columns ...string) bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(q *mysql.DeleteQuery) {
		q.SetTableAlias(alias, columns...)
	})
}

func Lateral() bob.Mod[*mysql.DeleteQuery] {
	return dialect.Lateral[*mysql.DeleteQuery]()
}

func UseIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.DeleteQuery] {
	return dialect.UseIndex[*mysql.DeleteQuery](first, others...)
}

func IgnoreIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.DeleteQuery] {
	return dialect.UseIndex[*mysql.DeleteQuery](first, others...)
}

func ForceIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.DeleteQuery] {
	return dialect.UseIndex[*mysql.DeleteQuery](first, others...)
}

func Partition(partitions ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Partition[*mysql.InsertQuery](partitions...)
}

func From(name any) bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(u *mysql.DeleteQuery) {
		u.Tables = append(u.Tables, clause.Table{
			Expression: name,
		})
	})
}

func FromAs(name any, alias string) bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(u *mysql.DeleteQuery) {
		u.Tables = append(u.Tables, clause.Table{
			Expression: name,
			Alias:      alias,
		})
	})
}

func Using(table any) bob.Mod[*mysql.DeleteQuery] {
	return mods.QueryModFunc[*mysql.DeleteQuery](func(q *mysql.DeleteQuery) {
		q.SetTable(table)
	})
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

func Where(e bob.Expression) bob.Mod[*mysql.DeleteQuery] {
	return mods.Where[*mysql.DeleteQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*mysql.DeleteQuery] {
	return mods.Where[*mysql.DeleteQuery]{mysql.Raw(clause, args...)}
}

func OrderBy(e any) dialect.OrderBy[*mysql.DeleteQuery] {
	return dialect.OrderBy[*mysql.DeleteQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*mysql.DeleteQuery] {
	return mods.Limit[*mysql.DeleteQuery]{
		Count: count,
	}
}
