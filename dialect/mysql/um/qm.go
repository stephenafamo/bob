package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
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

func LowPriority() bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(i dialect.UpdateEditor) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Ignore() bob.Mod[dialect.UpdateEditor] {
	return bob.ModFunc[dialect.UpdateEditor](func(i dialect.UpdateEditor) {
		i.AppendModifier("IGNORE")
	})
}

func Table(name any) dialect.FromChain[dialect.UpdateEditor] {
	return dialect.From[dialect.UpdateEditor](name)
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

func CrossJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.CrossJoin[dialect.UpdateEditor](e)
}

func StraightJoin(e any) dialect.JoinChain[dialect.UpdateEditor] {
	return dialect.StraightJoin[dialect.UpdateEditor](e)
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

// SetExpr is like SetCol but the column LHS is any expression (e.g. mysql.Quote("t", "col")).
func SetExpr(col bob.Expression) mods.Set[dialect.UpdateEditor] {
	return mods.Set[dialect.UpdateEditor]{Col: col}
}

func Where(e bob.Expression) mods.Where[dialect.UpdateEditor] {
	return mods.Where[dialect.UpdateEditor]{E: e}
}

func OrderBy(e any) dialect.OrderBy[dialect.UpdateEditor] {
	return dialect.OrderBy[dialect.UpdateEditor](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[dialect.UpdateEditor] {
	return mods.Limit[dialect.UpdateEditor]{
		Count: count,
	}
}
