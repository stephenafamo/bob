package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*mysql.UpdateQuery] {
	return dialect.With[*mysql.UpdateQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*mysql.UpdateQuery] {
	return mods.Recursive[*mysql.UpdateQuery](r)
}

func LowPriority() bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(i *mysql.UpdateQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func Ignore() bob.Mod[*mysql.UpdateQuery] {
	return mods.QueryModFunc[*mysql.UpdateQuery](func(i *mysql.UpdateQuery) {
		i.AppendModifier("IGNORE")
	})
}

func Table(name any) dialect.FromChain[*mysql.UpdateQuery] {
	return dialect.From[*mysql.UpdateQuery](name)
}

func Set(a string, b any) bob.Mod[*mysql.UpdateQuery] {
	return mods.Set[*mysql.UpdateQuery]{expr.OP("=", mysql.Quote(a), b)}
}

func SetArg(a string, b any) bob.Mod[*mysql.UpdateQuery] {
	return mods.Set[*mysql.UpdateQuery]{expr.OP("=", mysql.Quote(a), mysql.Arg(b))}
}

func Where(e bob.Expression) bob.Mod[*mysql.UpdateQuery] {
	return mods.Where[*mysql.UpdateQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*mysql.UpdateQuery] {
	return mods.Where[*mysql.UpdateQuery]{mysql.Raw(clause, args...)}
}

func OrderBy(e any) dialect.OrderBy[*mysql.UpdateQuery] {
	return dialect.OrderBy[*mysql.UpdateQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Limit(count int64) bob.Mod[*mysql.UpdateQuery] {
	return mods.Limit[*mysql.UpdateQuery]{
		Count: count,
	}
}
