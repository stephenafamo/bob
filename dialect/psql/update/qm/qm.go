package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*psql.UpdateQuery] {
	return dialect.With[*psql.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*psql.UpdateQuery] {
	return mods.Recursive[*psql.UpdateQuery](r)
}

func Only() bob.Mod[*psql.UpdateQuery] {
	return mods.QueryModFunc[*psql.UpdateQuery](func(u *psql.UpdateQuery) {
		u.Only = true
	})
}

func Table(name any) bob.Mod[*psql.UpdateQuery] {
	return mods.QueryModFunc[*psql.UpdateQuery](func(u *psql.UpdateQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func TableAs(name any, alias string) bob.Mod[*psql.UpdateQuery] {
	return mods.QueryModFunc[*psql.UpdateQuery](func(u *psql.UpdateQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func Set(a string, b any) bob.Mod[*psql.UpdateQuery] {
	return mods.Set[*psql.UpdateQuery]{expr.OP("=", psql.Quote(a), b)}
}

func SetArg(a string, b any) bob.Mod[*psql.UpdateQuery] {
	return mods.Set[*psql.UpdateQuery]{expr.OP("=", psql.Quote(a), psql.Arg(b))}
}

func From(table any) dialect.FromChain[*psql.UpdateQuery] {
	return dialect.From[*psql.UpdateQuery](table)
}

func FromFunction(funcs ...*dialect.Function) dialect.FromChain[*psql.UpdateQuery] {
	var table any

	if len(funcs) == 1 {
		table = funcs[0]
	}

	if len(funcs) > 1 {
		table = dialect.Functions(funcs)
	}

	return dialect.From[*psql.UpdateQuery](table)
}

func InnerJoin(e any) dialect.JoinChain[*psql.UpdateQuery] {
	return dialect.InnerJoin[*psql.UpdateQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*psql.UpdateQuery] {
	return dialect.LeftJoin[*psql.UpdateQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*psql.UpdateQuery] {
	return dialect.RightJoin[*psql.UpdateQuery](e)
}

func FullJoin(e any) dialect.JoinChain[*psql.UpdateQuery] {
	return dialect.FullJoin[*psql.UpdateQuery](e)
}

func CrossJoin(e any) bob.Mod[*psql.UpdateQuery] {
	return dialect.CrossJoin[*psql.UpdateQuery](e)
}

func Where(e bob.Expression) bob.Mod[*psql.UpdateQuery] {
	return mods.Where[*psql.UpdateQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*psql.UpdateQuery] {
	return mods.Where[*psql.UpdateQuery]{psql.Raw(clause, args...)}
}

func Returning(clauses ...any) bob.Mod[*psql.UpdateQuery] {
	return mods.Returning[*psql.UpdateQuery](clauses)
}
