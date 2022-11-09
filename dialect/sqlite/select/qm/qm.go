package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*sqlite.SelectQuery] {
	return dialect.With[*sqlite.SelectQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*sqlite.SelectQuery] {
	return mods.Recursive[*sqlite.SelectQuery](r)
}

func Distinct() bob.Mod[*sqlite.SelectQuery] {
	return mods.QueryModFunc[*sqlite.SelectQuery](func(q *sqlite.SelectQuery) {
		q.Distinct = true
	})
}

func Columns(clauses ...any) bob.Mod[*sqlite.SelectQuery] {
	return mods.Select[*sqlite.SelectQuery](clauses)
}

func From(table any) bob.Mod[*sqlite.SelectQuery] {
	return mods.QueryModFunc[*sqlite.SelectQuery](func(q *sqlite.SelectQuery) {
		q.SetTable(table)
	})
}

func As(alias string, columns ...string) bob.Mod[*sqlite.SelectQuery] {
	return dialect.As[*sqlite.SelectQuery](alias, columns...)
}

func NotIndexed() bob.Mod[*sqlite.SelectQuery] {
	return dialect.NotIndexed[*sqlite.SelectQuery]()
}

func IndexedBy(index string) bob.Mod[*sqlite.SelectQuery] {
	return dialect.IndexedBy[*sqlite.SelectQuery](index)
}

func InnerJoin(e any) dialect.JoinChain[*sqlite.SelectQuery] {
	return dialect.InnerJoin[*sqlite.SelectQuery](e)
}

func LeftJoin(e any) dialect.JoinChain[*sqlite.SelectQuery] {
	return dialect.LeftJoin[*sqlite.SelectQuery](e)
}

func RightJoin(e any) dialect.JoinChain[*sqlite.SelectQuery] {
	return dialect.RightJoin[*sqlite.SelectQuery](e)
}

func FullJoin(e any) dialect.JoinChain[*sqlite.SelectQuery] {
	return dialect.FullJoin[*sqlite.SelectQuery](e)
}

func CrossJoin(e any) bob.Mod[*sqlite.SelectQuery] {
	return dialect.CrossJoin[*sqlite.SelectQuery](e)
}

func Where(e bob.Expression) bob.Mod[*sqlite.SelectQuery] {
	return mods.Where[*sqlite.SelectQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*sqlite.SelectQuery] {
	return mods.Where[*sqlite.SelectQuery]{sqlite.Raw(clause, args...)}
}

func Having(e bob.Expression) bob.Mod[*sqlite.SelectQuery] {
	return mods.Having[*sqlite.SelectQuery]{e}
}

func HavingClause(clause string, args ...any) bob.Mod[*sqlite.SelectQuery] {
	return mods.Having[*sqlite.SelectQuery]{sqlite.Raw(clause, args...)}
}

func GroupBy(e any) bob.Mod[*sqlite.SelectQuery] {
	return mods.GroupBy[*sqlite.SelectQuery]{
		E: e,
	}
}

func Window(name string) dialect.WindowMod[*sqlite.SelectQuery] {
	m := dialect.WindowMod[*sqlite.SelectQuery]{
		Name: name,
	}

	m.WindowChain = &dialect.WindowChain[*dialect.WindowMod[*sqlite.SelectQuery]]{
		Wrap: &m,
	}
	return m
}

func OrderBy(e any) dialect.OrderBy[*sqlite.SelectQuery] {
	return dialect.OrderBy[*sqlite.SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// Sqlite can use an clauseession for the limit
func Limit(count any) bob.Mod[*sqlite.SelectQuery] {
	return mods.Limit[*sqlite.SelectQuery]{
		Count: count,
	}
}

// Sqlite can use an clauseession for the offset
func Offset(count any) bob.Mod[*sqlite.SelectQuery] {
	return mods.Offset[*sqlite.SelectQuery]{
		Count: count,
	}
}

func Union(q bob.Query) bob.Mod[*sqlite.SelectQuery] {
	return mods.Combine[*sqlite.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func UnionAll(q bob.Query) bob.Mod[*sqlite.SelectQuery] {
	return mods.Combine[*sqlite.SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func Intersect(q bob.Query) bob.Mod[*sqlite.SelectQuery] {
	return mods.Combine[*sqlite.SelectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func Except(q bob.Query) bob.Mod[*sqlite.SelectQuery] {
	return mods.Combine[*sqlite.SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}
