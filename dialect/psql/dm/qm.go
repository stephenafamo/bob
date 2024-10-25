package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.DeleteQuery] {
	return dialect.With[*dialect.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.DeleteQuery] {
	return mods.Recursive[*dialect.DeleteQuery](r)
}

func Only() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(d *dialect.DeleteQuery) {
		d.Only = true
	})
}

func From(name any) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(u *dialect.DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func FromAs(name any, alias string) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(u *dialect.DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func Using(table any) dialect.FromChain[*dialect.DeleteQuery] {
	return dialect.From[*dialect.DeleteQuery](table)
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

func FullJoin(e any) dialect.JoinChain[*dialect.DeleteQuery] {
	return dialect.FullJoin[*dialect.DeleteQuery](e)
}

func CrossJoin(e any) dialect.CrossJoinChain[*dialect.DeleteQuery] {
	return dialect.CrossJoin[*dialect.DeleteQuery](e)
}

func Where(e bob.Expression) mods.Where[*dialect.DeleteQuery] {
	return mods.Where[*dialect.DeleteQuery]{E: e}
}

func Returning(clauses ...any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Returning[*dialect.DeleteQuery](clauses)
}
