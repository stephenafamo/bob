package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*mysql.UpdateQuery] {
	return dialect.With[*mysql.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*mysql.UpdateQuery] {
	return mods.Recursive[*mysql.UpdateQuery](r)
}

func LowPriority() bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(i *mysql.UpdateQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Ignore() bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(i *mysql.UpdateQuery) {
		i.AppendModifier("IGNORE")
	})
}

func Table(name any) bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(u *mysql.UpdateQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func TableAs(name any, alias string) bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(u *mysql.UpdateQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func As(alias string, columns ...string) bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(q *mysql.UpdateQuery) {
		q.SetTableAlias(alias, columns...)
	})
}

func Lateral() bob.Mod[*mysql.UpdateQuery] {
	return dialect.Lateral[*mysql.UpdateQuery]()
}

func UseIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.UpdateQuery] {
	return dialect.UseIndex[*mysql.UpdateQuery](first, others...)
}

func IgnoreIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.UpdateQuery] {
	return dialect.UseIndex[*mysql.UpdateQuery](first, others...)
}

func ForceIndex(first string, others ...string) *dialect.IndexHintChain[*mysql.UpdateQuery] {
	return dialect.UseIndex[*mysql.UpdateQuery](first, others...)
}

func Partition(partitions ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Partition[*mysql.InsertQuery](partitions...)
}

func Set(a string, b any) bob.Mod[*mysql.UpdateQuery] {
	return mods.Set[*mysql.UpdateQuery]{expr.OP("=", mysql.Quote(a), b)}
}

func SetArg(a string, b any) bob.Mod[*mysql.UpdateQuery] {
	return mods.Set[*mysql.UpdateQuery]{expr.OP("=", mysql.Quote(a), mysql.Arg(b))}
}

func Where(e bob.Expression) bob.Mod[*mysql.UpdateQuery] {
	return mods.Where[*mysql.UpdateQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*mysql.UpdateQuery] {
	return mods.Where[*mysql.UpdateQuery]{mysql.Raw(clause, args...)}
}

func OrderBy(e any) dialect.OrderBy[*mysql.UpdateQuery] {
	return dialect.OrderBy[*mysql.UpdateQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*mysql.UpdateQuery] {
	return mods.Limit[*mysql.UpdateQuery]{
		Count: count,
	}
}
