package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.DeleteQuery] {
	return dialect.With[*dialect.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.DeleteQuery] {
	return mods.Recursive[*dialect.DeleteQuery](r)
}

func LowPriority() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(i *dialect.DeleteQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Quick() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(i *dialect.DeleteQuery) {
		i.AppendModifier("QUICK")
	})
}

func Ignore() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(i *dialect.DeleteQuery) {
		i.AppendModifier("IGNORE")
	})
}

func From(name any, partitions ...string) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(u *dialect.DeleteQuery) {
		u.Tables = append(u.Tables, clause.TableRef{
			Expression: name,
		})
		u.Partitions = partitions
	})
}

func FromAs(name any, alias string, partitions ...string) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(u *dialect.DeleteQuery) {
		u.Tables = append(u.Tables, clause.TableRef{
			Expression: name,
			Alias:      alias,
		})
		u.Partitions = partitions
	})
}

func Using(name any) dialect.FromChain[*dialect.DeleteQuery] {
	return dialect.From[*dialect.DeleteQuery](name)
}

func InnerJoin(e any) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.InnerJoin[*dialect.DeleteQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.LeftJoin[*dialect.DeleteQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.RightJoin[*dialect.DeleteQuery](e)
}

func CrossJoin(e any) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.CrossJoin[*dialect.DeleteQuery](e)
}

func StraightJoin(e any) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.StraightJoin[*dialect.DeleteQuery](e)
}

func Where(e bob.Expression) mods.Where[*dialect.DeleteQuery] {
	return mods.Where[*dialect.DeleteQuery]{E: e}
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
