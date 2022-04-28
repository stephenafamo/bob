package psql

import (
	"io"

	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Update(mods ...mods.QueryMod[*UpdateQuery]) *UpdateQuery {
	s := &UpdateQuery{}
	for _, mod := range mods {
		mod.Apply(s)
	}

	return s
}

// Not handling on-conflict yet
type UpdateQuery struct {
	expr.With
	only bool
	expr.Table
	expr.Set
	expr.FromItems
	expr.Where
	expr.Returning
}

func (u UpdateQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return u.WriteSQL(w, dialect, start)
}

func (u UpdateQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	tableArgs, err := query.ExpressIf(w, d, start+len(args), u.Table, true, "UPDATE ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	setArgs, err := query.ExpressIf(w, d, start+len(args), u.Set, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	fromArgs, err := query.ExpressSlice(w, d, start+len(args), u.FromItems.Items, "\nFROM ", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := query.ExpressIf(w, d, start+len(args), u.Where,
		len(u.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := query.ExpressIf(w, d, start+len(args), u.Returning,
		len(u.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}

type UpdateQM struct {
	withMod[*UpdateQuery]
	fromMod[*UpdateQuery]
	fromItemMod
	joinMod[*expr.FromItem]
}

func (qm UpdateQM) Only() mods.QueryMod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(u *UpdateQuery) {
		u.only = true
	})
}

func (qm UpdateQM) Table(name any) mods.QueryMod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(u *UpdateQuery) {
		u.Table = expr.Table{
			Expression: name,
		}
	})
}

func (qm UpdateQM) TableAs(name any, alias string) mods.QueryMod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(u *UpdateQuery) {
		u.Table = expr.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm UpdateQM) Set(a, b any) mods.QueryMod[*UpdateQuery] {
	return mods.Set[*UpdateQuery]{expr.EQ(a, b)}
}

func (qm UpdateQM) SetArg(a, b any) mods.QueryMod[*UpdateQuery] {
	return mods.Set[*UpdateQuery]{expr.EQ(a, expr.Arg(b))}
}

func (qm UpdateQM) Where(e query.Expression) mods.QueryMod[*UpdateQuery] {
	return mods.Where[*UpdateQuery]{e}
}

func (qm UpdateQM) WhereClause(clause string, args ...any) mods.QueryMod[*UpdateQuery] {
	return mods.Where[*UpdateQuery]{expr.Statement(clause, args...)}
}

func (qm UpdateQM) Returning(expressions ...any) mods.QueryMod[*UpdateQuery] {
	return mods.Returning[*UpdateQuery](expressions)
}
