package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[dialect.DeleteEditor] {
	return dialect.With[dialect.DeleteEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.DeleteEditor] {
	return mods.Recursive[dialect.DeleteEditor](r)
}

func LowPriority() bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(i dialect.DeleteEditor) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Quick() bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(i dialect.DeleteEditor) {
		i.AppendModifier("QUICK")
	})
}

func Ignore() bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(i dialect.DeleteEditor) {
		i.AppendModifier("IGNORE")
	})
}

func From(name any, partitions ...string) bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(u dialect.DeleteEditor) {
		u.AppendDeleteTable(clause.TableRef{
			Expression: name,
		})
		u.SetPartitions(partitions...)
	})
}

func FromAs(name any, alias string, partitions ...string) bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(u dialect.DeleteEditor) {
		u.AppendDeleteTable(clause.TableRef{
			Expression: name,
			Alias:      alias,
		})
		u.SetPartitions(partitions...)
	})
}

func Using(name any) dialect.FromChain[dialect.DeleteEditor] {
	return dialect.From[dialect.DeleteEditor](name)
}

func InnerJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.InnerJoin[dialect.DeleteEditor](e)
}

func LeftJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.LeftJoin[dialect.DeleteEditor](e)
}

func RightJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.RightJoin[dialect.DeleteEditor](e)
}

func CrossJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.CrossJoin[dialect.DeleteEditor](e)
}

func StraightJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.StraightJoin[dialect.DeleteEditor](e)
}

func Where(e bob.Expression) mods.Where[dialect.DeleteEditor] {
	return mods.Where[dialect.DeleteEditor]{E: e}
}

func OrderBy(e any) dialect.OrderBy[dialect.DeleteEditor] {
	return dialect.OrderBy[dialect.DeleteEditor](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[dialect.DeleteEditor] {
	return mods.Limit[dialect.DeleteEditor]{
		Count: count,
	}
}
