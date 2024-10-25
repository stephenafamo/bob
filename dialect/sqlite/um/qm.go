package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.UpdateQuery] {
	return dialect.With[*dialect.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.UpdateQuery] {
	return mods.Recursive[*dialect.UpdateQuery](r)
}

func OrAbort() bob.Mod[*dialect.UpdateQuery] {
	return dialect.OrAbort[*dialect.UpdateQuery]()
}

func OrFail() bob.Mod[*dialect.UpdateQuery] {
	return dialect.OrFail[*dialect.UpdateQuery]()
}

func OrIgnore() bob.Mod[*dialect.UpdateQuery] {
	return dialect.OrIgnore[*dialect.UpdateQuery]()
}

func OrReplace() bob.Mod[*dialect.UpdateQuery] {
	return dialect.OrReplace[*dialect.UpdateQuery]()
}

func OrRollback() bob.Mod[*dialect.UpdateQuery] {
	return dialect.OrRollback[*dialect.UpdateQuery]()
}

func Table(name any) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.Table.Table = name
	})
}

func TableAs(name any, alias string) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.Table.Table = name
		q.Table.Alias = alias
	})
}

func TableIndexedBy(i string) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.Table.IndexedBy = &i
	})
}

func TableNotIndexed() bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		var s string
		q.Table.IndexedBy = &s
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
