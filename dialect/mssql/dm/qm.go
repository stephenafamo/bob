package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mssql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.DeleteQuery] {
	return dialect.With[*dialect.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.DeleteQuery] {
	return mods.Recursive[*dialect.DeleteQuery](r)
}

func From(name any) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(d *dialect.DeleteQuery) {
		d.Table = clause.TableRef{
			Expression: name,
		}
	})
}

func FromAs(name any, alias string) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(d *dialect.DeleteQuery) {
		d.Table = clause.TableRef{
			Expression: name,
			Alias:      alias,
		}
	})
}

// Top limits the number of rows deleted
// SQL: DELETE TOP (count) FROM ...
func Top(count any) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(d *dialect.DeleteQuery) {
		d.Top = &dialect.Top{Count: count}
	})
}

// Using adds a FROM clause for joins in DELETE statements
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

// Output specifies the OUTPUT clause for DELETE
// SQL: DELETE FROM ... OUTPUT deleted.col1, deleted.col2 ...
func Output(clauses ...any) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(d *dialect.DeleteQuery) {
		d.AppendOutput(clauses...)
	})
}
