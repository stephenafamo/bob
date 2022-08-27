package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	pmods "github.com/stephenafamo/bob/dialect/psql/mods"
	"github.com/stephenafamo/bob/mods"
)

// type deleteQM struct {
// joinMod[*clause.From]
// }

func With(name string, columns ...string) pmods.CteChain[*psql.DeleteQuery] {
	return pmods.With[*psql.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*psql.DeleteQuery] {
	return mods.Recursive[*psql.DeleteQuery](r)
}

func Only() bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(d *psql.DeleteQuery) {
		d.Only = true
	})
}

func From(name any) bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(u *psql.DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func FromAs(name any, alias string) bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(u *psql.DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func Using(table any) bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(q *psql.DeleteQuery) {
		q.SetTable(table)
	})
}

func UsingAs(alias string, columns ...string) bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(q *psql.DeleteQuery) {
		q.SetTableAlias(alias, columns...)
	})
}

func UsingOnly() bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(q *psql.DeleteQuery) {
		q.SetOnly(true)
	})
}

func Lateral() bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(q *psql.DeleteQuery) {
		q.SetLateral(true)
	})
}

func WithOrdinality() bob.Mod[*psql.DeleteQuery] {
	return mods.QueryModFunc[*psql.DeleteQuery](func(q *psql.DeleteQuery) {
		q.SetWithOrdinality(true)
	})
}

func InnerJoin(e any) pmods.JoinChain[*psql.DeleteQuery] {
	return pmods.InnerJoin[*psql.DeleteQuery](e)
}

func LeftJoin(e any) pmods.JoinChain[*psql.DeleteQuery] {
	return pmods.LeftJoin[*psql.DeleteQuery](e)
}

func RightJoin(e any) pmods.JoinChain[*psql.DeleteQuery] {
	return pmods.RightJoin[*psql.DeleteQuery](e)
}

func FullJoin(e any) pmods.JoinChain[*psql.DeleteQuery] {
	return pmods.FullJoin[*psql.DeleteQuery](e)
}

func CrossJoin(e any) bob.Mod[*psql.DeleteQuery] {
	return pmods.CrossJoin[*psql.DeleteQuery](e)
}

func Where(e bob.Expression) bob.Mod[*psql.DeleteQuery] {
	return mods.Where[*psql.DeleteQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*psql.DeleteQuery] {
	return mods.Where[*psql.DeleteQuery]{psql.Raw(clause, args...)}
}

func Returning(clauses ...any) bob.Mod[*psql.DeleteQuery] {
	return mods.Returning[*psql.DeleteQuery](clauses)
}
