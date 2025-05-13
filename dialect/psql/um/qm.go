package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.UpdateQuery] {
	return dialect.With[*dialect.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.UpdateQuery] {
	return mods.Recursive[*dialect.UpdateQuery](r)
}

func Only() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(u *dialect.UpdateQuery) {
		u.Only = true
	})
}

func Table(name any) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(u *dialect.UpdateQuery) {
		u.Table = clause.TableRef{
			Expression: name,
		}
	})
}

func TableAs(name any, alias string) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(u *dialect.UpdateQuery) {
		u.Table = clause.TableRef{
			Expression: name,
			Alias:      alias,
		}
	})
}

func Set(sets ...bob.Expression) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.Set.Set = append(q.Set.Set, internal.ToAnySlice(sets)...)
	})
}

func SetCol(from string) mods.Set[*dialect.UpdateQuery] {
	return mods.Set[*dialect.UpdateQuery]([]string{from})
}

func From(table any) dialect.FromChain[*dialect.UpdateQuery] {
	return dialect.From[*dialect.UpdateQuery](table)
}

func FromFunction(funcs ...*dialect.Function) dialect.FromChain[*dialect.UpdateQuery] {
	var table any

	if len(funcs) == 1 {
		table = funcs[0]
	}

	if len(funcs) > 1 {
		table = dialect.Functions(funcs)
	}

	return dialect.From[*dialect.UpdateQuery](table)
}

func InnerJoin(e any) dialect.JoinChain[*dialect.UpdateQuery] {
	return dialect.InnerJoin[*dialect.UpdateQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*dialect.UpdateQuery] {
	return dialect.LeftJoin[*dialect.UpdateQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*dialect.UpdateQuery] {
	return dialect.RightJoin[*dialect.UpdateQuery](e)
}

func FullJoin(e any) dialect.JoinChain[*dialect.UpdateQuery] {
	return dialect.FullJoin[*dialect.UpdateQuery](e)
}

func CrossJoin(e any) dialect.CrossJoinChain[*dialect.UpdateQuery] {
	return dialect.CrossJoin[*dialect.UpdateQuery](e)
}

func Where(e bob.Expression) mods.Where[*dialect.UpdateQuery] {
	return mods.Where[*dialect.UpdateQuery]{E: e}
}

func Returning(clauses ...any) bob.Mod[*dialect.UpdateQuery] {
	return mods.Returning[*dialect.UpdateQuery](clauses)
}
