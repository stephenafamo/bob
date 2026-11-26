package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[dialect.UpdateEditor] {
	return dialect.With[dialect.UpdateEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.UpdateEditor] {
	return mods.Recursive[dialect.UpdateEditor](r)
}

func OrAbort() bob.Mod[dialect.UpdateEditor] {
	return dialect.OrAbort[dialect.UpdateEditor]()
}

func OrFail() bob.Mod[dialect.UpdateEditor] {
	return dialect.OrFail[dialect.UpdateEditor]()
}

func OrIgnore() bob.Mod[dialect.UpdateEditor] {
	return dialect.OrIgnore[dialect.UpdateEditor]()
}

func OrReplace() bob.Mod[dialect.UpdateEditor] {
	return dialect.OrReplace[dialect.UpdateEditor]()
}

func OrRollback() bob.Mod[dialect.UpdateEditor] {
	return dialect.OrRollback[dialect.UpdateEditor]()
}

func Table(name any) bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
		q.SetTargetName(name)
	})
}

func TableAs(name any, alias string) bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
		q.SetTargetName(name)
		q.SetTargetAlias(alias)
	})
}

func IndexedBy(i string) bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
		q.SetTargetIndexedBy(&i)
	})
}

func NotIndexed() bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
		var s string
		q.SetTargetIndexedBy(&s)
	})
}

func Set(sets ...bob.Expression) bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
		q.AppendSet(internal.ToAnySlice(sets)...)
	})
}

// SetCol sets one column in UPDATE ... SET. The column name is quoted automatically.
// For qualified names or other expressions on the LHS, use SetExpr.
func SetCol(from string) mods.Set[dialect.UpdateEditor] {
	return mods.Set[dialect.UpdateEditor]{Col: expr.Quote(from)}
}

// SetExpr is like SetCol but the column LHS is any expression (e.g. sqlite.Quote("t", "col")).
func SetExpr(col bob.Expression) mods.Set[dialect.UpdateEditor] {
	return mods.Set[dialect.UpdateEditor]{Col: col}
}

// SetCols creates a multi-column setter: (columns...) = (values...) | (subquery)
func SetCols(columns ...string) clause.SetCols[dialect.UpdateEditor] {
	return clause.NewSetCols[dialect.UpdateEditor](columns...)
}

func From(table any) dialect.FromChain[dialect.UpdateEditor] {
	return dialect.From[dialect.UpdateEditor](table)
}

func InnerJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.InnerJoin[dialect.UpdateEditor](e)
}

func LeftJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.LeftJoin[dialect.UpdateEditor](e)
}

func RightJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.RightJoin[dialect.UpdateEditor](e)
}

func FullJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.FullJoin[dialect.UpdateEditor](e)
}

func CrossJoin(e any) dialect.CrossJoinChain[dialect.UpdateEditor] {
	return dialect.CrossJoin[dialect.UpdateEditor](e)
}

func Where(e bob.Expression) mods.Where[dialect.UpdateEditor] {
	return mods.Where[dialect.UpdateEditor]{E: e}
}

func Returning(clauses ...any) bob.Mod[dialect.UpdateEditor] {
	return mods.Returning[dialect.UpdateEditor](clauses)
}

func Limit(count any) bob.Mod[dialect.UpdateEditor] {
	return mods.Limit[dialect.UpdateEditor]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[dialect.UpdateEditor] {
	return mods.Offset[dialect.UpdateEditor]{
		Count: count,
	}
}
