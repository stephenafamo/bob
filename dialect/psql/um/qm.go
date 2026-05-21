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

// SetCols creates a multi-column setter: (columns...) = ROW(...) | (values...) | (subquery)
func SetCols(columns ...string) dialect.SetCols[*dialect.UpdateQuery] {
	return dialect.NewSetCols[*dialect.UpdateQuery](columns...)
}

func From(table any, joins ...dialect.JoinChain[*dialect.UpdateQuery]) dialect.FromChain[*dialect.UpdateQuery] {
	return dialect.From[*dialect.UpdateQuery](table, joins...)
}

// FromFunction returns an expression for um.From when the source is one or more
// table functions (ROWS FROM when multiple).
func FromFunction(funcs ...*dialect.Function) bob.Expression {
	return dialect.TableFunctions(funcs...)
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

func CrossJoin(e any) dialect.JoinChain[*dialect.UpdateQuery] {
	return dialect.CrossJoin[*dialect.UpdateQuery](e)
}

func Where(e bob.Expression) mods.Where[*dialect.UpdateQuery] {
	return mods.Where[*dialect.UpdateQuery]{E: e}
}

func WhereCurrentOf(cursor string) mods.WhereCurrentOf[*dialect.UpdateQuery] {
	return mods.WhereCurrentOf[*dialect.UpdateQuery]{Cursor: cursor}
}

func Returning(clauses ...any) mods.Returning[*dialect.UpdateQuery] {
	return mods.Returning[*dialect.UpdateQuery](clauses)
}
