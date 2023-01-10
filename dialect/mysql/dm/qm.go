package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.DeleteQuery] {
	return dialect.With[*dialect.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.DeleteQuery] {
	return mods.Recursive[*dialect.DeleteQuery](r)
}

func LowPriority() bob.Mod[*dialect.DeleteQuery] {
	return mods.QueryModFunc[*dialect.DeleteQuery](func(i *dialect.DeleteQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Quick() bob.Mod[*dialect.DeleteQuery] {
	return mods.QueryModFunc[*dialect.DeleteQuery](func(i *dialect.DeleteQuery) {
		i.AppendModifier("QUICK")
	})
}

func Ignore() bob.Mod[*dialect.DeleteQuery] {
	return mods.QueryModFunc[*dialect.DeleteQuery](func(i *dialect.DeleteQuery) {
		i.AppendModifier("IGNORE")
	})
}

func From(name any, partitions ...string) bob.Mod[*dialect.DeleteQuery] {
	return mods.QueryModFunc[*dialect.DeleteQuery](func(u *dialect.DeleteQuery) {
		u.Tables = append(u.Tables, clause.Table{
			Expression: name,
			Partitions: partitions,
		})
	})
}

func FromAs(name any, alias string, partitions ...string) bob.Mod[*dialect.DeleteQuery] {
	return mods.QueryModFunc[*dialect.DeleteQuery](func(u *dialect.DeleteQuery) {
		u.Tables = append(u.Tables, clause.Table{
			Expression: name,
			Alias:      alias,
			Partitions: partitions,
		})
	})
}

func Using(name any) dialect.FromChain[*dialect.DeleteQuery] {
	return dialect.From[*dialect.DeleteQuery](name)
}

func InnerJoin(e bob.Expression) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.InnerJoin[*dialect.DeleteQuery](e)
}

func LeftJoin(e bob.Expression) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.LeftJoin[*dialect.DeleteQuery](e)
}

func RightJoin(e bob.Expression) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.RightJoin[*dialect.DeleteQuery](e)
}

func CrossJoin(e bob.Expression) bob.Mod[*dialect.DeleteQuery] {
	return dialect.CrossJoin[*dialect.DeleteQuery](e)
}

func StraightJoin(e bob.Expression) bob.Mod[*dialect.DeleteQuery] {
	return dialect.StraightJoin[*dialect.DeleteQuery](e)
}

func Where(e bob.Expression) bob.Mod[*dialect.DeleteQuery] {
	return mods.Where[*dialect.DeleteQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Where[*dialect.DeleteQuery]{expr.RawQuery(dialect.Dialect, clause, args...)}
}

func OrderBy(e any) dialect.OrderBy[*dialect.DeleteQuery] {
	return dialect.OrderBy[*dialect.DeleteQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*dialect.DeleteQuery] {
	return mods.Limit[*dialect.DeleteQuery]{
		Count: count,
	}
}
