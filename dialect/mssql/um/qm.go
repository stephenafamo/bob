package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mssql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.UpdateQuery] {
	return dialect.With[*dialect.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.UpdateQuery] {
	return mods.Recursive[*dialect.UpdateQuery](r)
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

// Top limits the number of rows updated
// SQL: UPDATE TOP (count) ...
func Top(count any) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(u *dialect.UpdateQuery) {
		u.Top = &dialect.Top{Count: count}
	})
}

func Set(sets ...bob.Expression) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
		q.Set.Set = append(q.Set.Set, internal.ToAnySlice(sets)...)
	})
}

// SetCol sets one column in UPDATE ... SET. The column name is quoted automatically.
func SetCol(from string) mods.Set[*dialect.UpdateQuery] {
	return mods.Set[*dialect.UpdateQuery]{Col: expr.Quote(from)}
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

// Output specifies the OUTPUT clause for UPDATE
// SQL: UPDATE ... SET ... OUTPUT inserted.col1, deleted.col2 ...
func Output(clauses ...any) bob.Mod[*dialect.UpdateQuery] {
	return bob.ModFunc[*dialect.UpdateQuery](func(u *dialect.UpdateQuery) {
		u.AppendOutput(clauses...)
	})
}
