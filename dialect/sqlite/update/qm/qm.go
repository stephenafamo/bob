package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*sqlite.UpdateQuery] {
	return dialect.With[*sqlite.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*sqlite.UpdateQuery] {
	return mods.Recursive[*sqlite.UpdateQuery](r)
}

func OrAbort() bob.Mod[*sqlite.UpdateQuery] {
	return dialect.OrAbort[*sqlite.UpdateQuery]()
}

func OrFail() bob.Mod[*sqlite.UpdateQuery] {
	return dialect.OrFail[*sqlite.UpdateQuery]()
}

func OrIgnore() bob.Mod[*sqlite.UpdateQuery] {
	return dialect.OrIgnore[*sqlite.UpdateQuery]()
}

func OrReplace() bob.Mod[*sqlite.UpdateQuery] {
	return dialect.OrReplace[*sqlite.UpdateQuery]()
}

func OrRollback() bob.Mod[*sqlite.UpdateQuery] {
	return dialect.OrRollback[*sqlite.UpdateQuery]()
}

func Table(name any) bob.Mod[*sqlite.UpdateQuery] {
	return mods.QueryModFunc[*sqlite.UpdateQuery](func(q *sqlite.UpdateQuery) {
		q.Table.Table = name
	})
}

func TableAs(name any, alias string) bob.Mod[*sqlite.UpdateQuery] {
	return mods.QueryModFunc[*sqlite.UpdateQuery](func(q *sqlite.UpdateQuery) {
		q.Table.Table = name
		q.Table.Alias = alias
	})
}

func TableIndexedBy(i string) bob.Mod[*sqlite.UpdateQuery] {
	return mods.QueryModFunc[*sqlite.UpdateQuery](func(q *sqlite.UpdateQuery) {
		q.Table.IndexedBy = &i
	})
}

func TableNotIndexed() bob.Mod[*sqlite.UpdateQuery] {
	return mods.QueryModFunc[*sqlite.UpdateQuery](func(q *sqlite.UpdateQuery) {
		var s string
		q.Table.IndexedBy = &s
	})
}

func Set(a string, b any) bob.Mod[*sqlite.UpdateQuery] {
	return mods.Set[*sqlite.UpdateQuery]{expr.OP("=", sqlite.Quote(a), b)}
}

func SetArg(a string, b any) bob.Mod[*sqlite.UpdateQuery] {
	return mods.Set[*sqlite.UpdateQuery]{expr.OP("=", sqlite.Quote(a), sqlite.Arg(b))}
}

func From(table any) bob.Mod[*sqlite.UpdateQuery] {
	return mods.QueryModFunc[*sqlite.UpdateQuery](func(q *sqlite.UpdateQuery) {
		q.SetTable(table)
	})
}

func FromAlias(alias string, columns ...string) bob.Mod[*sqlite.UpdateQuery] {
	return dialect.As[*sqlite.UpdateQuery](alias, columns...)
}

func FromNotIndexed() bob.Mod[*sqlite.UpdateQuery] {
	return dialect.NotIndexed[*sqlite.UpdateQuery]()
}

func FromIndexedBy(index string) bob.Mod[*sqlite.UpdateQuery] {
	return dialect.IndexedBy[*sqlite.UpdateQuery](index)
}

func InnerJoin(e any) dialect.JoinChain[*sqlite.UpdateQuery] {
	return dialect.InnerJoin[*sqlite.UpdateQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*sqlite.UpdateQuery] {
	return dialect.LeftJoin[*sqlite.UpdateQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*sqlite.UpdateQuery] {
	return dialect.RightJoin[*sqlite.UpdateQuery](e)
}

func FullJoin(e any) dialect.JoinChain[*sqlite.UpdateQuery] {
	return dialect.FullJoin[*sqlite.UpdateQuery](e)
}

func CrossJoin(e any) bob.Mod[*sqlite.UpdateQuery] {
	return dialect.CrossJoin[*sqlite.UpdateQuery](e)
}

func Where(e bob.Expression) bob.Mod[*sqlite.UpdateQuery] {
	return mods.Where[*sqlite.UpdateQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*sqlite.UpdateQuery] {
	return mods.Where[*sqlite.UpdateQuery]{sqlite.Raw(clause, args...)}
}

func Returning(clauses ...any) bob.Mod[*sqlite.UpdateQuery] {
	return mods.Returning[*sqlite.UpdateQuery](clauses)
}
