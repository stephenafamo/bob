package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

// With starts a CTE. The name and column list are quoted as SQL identifiers.
func With(name string, columns ...string) dialect.CTEChain[dialect.DeleteEditor] {
	return dialect.With[dialect.DeleteEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.DeleteEditor] {
	return mods.Recursive[dialect.DeleteEditor](r)
}

func Only() bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(d dialect.DeleteEditor) {
		d.SetOnly(true)
	})
}

func From(name any) bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(u dialect.DeleteEditor) {
		u.SetTarget(clause.TableRef{
			Expression: name,
		})
	})
}

func FromAs(name any, alias string) bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(u dialect.DeleteEditor) {
		u.SetTarget(clause.TableRef{
			Expression: name,
			Alias:      alias,
		})
	})
}

func Using(table any, joins ...dialect.JoinChain[dialect.DeleteEditor]) dialect.FromChain[dialect.DeleteEditor] {
	return dialect.From(table, joins...)
}

// UsingFunction returns an expression for dm.Using when the source is one or more
// table functions (ROWS FROM when multiple).
func UsingFunction(funcs ...*dialect.Function) bob.Expression {
	return dialect.TableFunctions(funcs...)
}

func InnerJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.InnerJoin[dialect.DeleteEditor](e)
}

func LeftJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.LeftJoin[dialect.DeleteEditor](e)
}

func RightJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.RightJoin[dialect.DeleteEditor](e)
}

func FullJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.FullJoin[dialect.DeleteEditor](e)
}

func CrossJoin(e any) dialect.JoinChain[dialect.DeleteEditor] {
	return dialect.CrossJoin[dialect.DeleteEditor](e)
}

func Where(e bob.Expression) mods.Where[dialect.DeleteEditor] {
	return mods.Where[dialect.DeleteEditor]{E: e}
}

func WhereCurrentOf(cursor string) mods.WhereCurrentOf[dialect.DeleteEditor] {
	return mods.WhereCurrentOf[dialect.DeleteEditor]{Cursor: cursor}
}

func Returning(clauses ...any) mods.Returning[dialect.DeleteEditor] {
	return mods.Returning[dialect.DeleteEditor](clauses)
}
