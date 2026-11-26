package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

// With starts a CTE. The name and column list are quoted as SQL identifiers.
func With(name string, columns ...string) dialect.CTEChain[dialect.UpdateEditor] {
	return dialect.With[dialect.UpdateEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.UpdateEditor] {
	return mods.Recursive[dialect.UpdateEditor](r)
}

func Only() bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(u dialect.UpdateEditor) {
		u.SetOnly(true)
	})
}

func Table(name any) bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(u dialect.UpdateEditor) {
		u.SetTarget(clause.TableRef{
			Expression: name,
		})
	})
}

func TableAs(name any, alias string) bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(u dialect.UpdateEditor) {
		u.SetTarget(clause.TableRef{
			Expression: name,
			Alias:      alias,
		})
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

// SetExpr is like SetCol but the column LHS is any expression (e.g. psql.Quote("t", "col")).
func SetExpr(col bob.Expression) mods.Set[dialect.UpdateEditor] {
	return mods.Set[dialect.UpdateEditor]{Col: col}
}

// SetCols creates a multi-column setter: (columns...) = ROW(...) | (values...) | (subquery)
func SetCols(columns ...string) clause.SetCols[dialect.UpdateEditor] {
	return clause.NewSetCols[dialect.UpdateEditor](columns...).Options(clause.SetColsOptions{RowPrefix: "ROW"})
}

func From(table any, joins ...dialect.JoinChain[dialect.UpdateEditor]) dialect.FromChain[dialect.UpdateEditor] {
	return dialect.From(table, joins...)
}

// FromFunction returns an expression for um.From when the source is one or more
// table functions (ROWS FROM when multiple).
func FromFunction(funcs ...*dialect.Function) bob.Expression {
	return dialect.TableFunctions(funcs...)
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

func CrossJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.CrossJoin[dialect.UpdateEditor](e)
}

func Where(e bob.Expression) mods.Where[dialect.UpdateEditor] {
	return mods.Where[dialect.UpdateEditor]{E: e}
}

func WhereCurrentOf(cursor string) mods.WhereCurrentOf[dialect.UpdateEditor] {
	return mods.WhereCurrentOf[dialect.UpdateEditor]{Cursor: cursor}
}

func Returning(clauses ...any) mods.Returning[dialect.UpdateEditor] {
	return mods.Returning[dialect.UpdateEditor](clauses)
}
