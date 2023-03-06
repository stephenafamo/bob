package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.UpdateQuery] {
	return dialect.With[*dialect.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.UpdateQuery] {
	return mods.Recursive[*dialect.UpdateQuery](r)
}

func LowPriority() bob.Mod[*dialect.UpdateQuery] {
	return mods.QueryModFunc[*dialect.UpdateQuery](func(i *dialect.UpdateQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Ignore() bob.Mod[*dialect.UpdateQuery] {
	return mods.QueryModFunc[*dialect.UpdateQuery](func(i *dialect.UpdateQuery) {
		i.AppendModifier("IGNORE")
	})
}

func Table(name any) dialect.FromChain[*dialect.UpdateQuery] {
	return dialect.From[*dialect.UpdateQuery](name)
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

func CrossJoin(e any) bob.Mod[*dialect.UpdateQuery] {
	return dialect.CrossJoin[*dialect.UpdateQuery](e)
}

func StraightJoin(e any) bob.Mod[*dialect.UpdateQuery] {
	return dialect.StraightJoin[*dialect.UpdateQuery](e)
}

func Set(a string, b any) bob.Mod[*dialect.UpdateQuery] {
	return mods.Set[*dialect.UpdateQuery]{expr.OP("=", expr.Quote(a), b)}
}

func SetArg(a string, b any) bob.Mod[*dialect.UpdateQuery] {
	return mods.Set[*dialect.UpdateQuery]{expr.OP("=", expr.Quote(a), expr.Arg(b))}
}

func Where(e any) bob.Mod[*dialect.UpdateQuery] {
	return mods.Where[*dialect.UpdateQuery]{e}
}

func OrderBy(e any) dialect.OrderBy[*dialect.UpdateQuery] {
	return dialect.OrderBy[*dialect.UpdateQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*dialect.UpdateQuery] {
	return mods.Limit[*dialect.UpdateQuery]{
		Count: count,
	}
}
