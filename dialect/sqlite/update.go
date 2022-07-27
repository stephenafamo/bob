package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/builder"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Update(queryMods ...query.Mod[*UpdateQuery]) query.BaseQuery[*UpdateQuery] {
	q := &UpdateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*UpdateQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_update.html
type UpdateQuery struct {
	expr.With
	or
	expr.FromItem
	expr.Set
	expr.FromItems
	expr.Where
	expr.Returning
}

func (u UpdateQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("UPDATE"))

	_, err = query.ExpressIf(w, d, start+len(args), u.or, true, " ", "")
	if err != nil {
		return nil, err
	}

	tableArgs, err := query.ExpressIf(w, d, start+len(args), u.FromItem, true, " ", "")
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
	builderMod
	withMod[*UpdateQuery]      // For CTEs
	mods.FromMod[*UpdateQuery] // update *FROM*
	joinMod[*expr.FromItem]    // joins, which are mods of the FROM
	fromItemMod                // Dialect specific fromItem mods
	orMod[*UpdateQuery]        // UPDATE or REPLACE|ABORT|IGNORE e.t.c.
}

func (qm UpdateQM) Table(name any) query.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.Table = expr.Table{
			Expression: name,
		}
	})
}

func (qm UpdateQM) TableAs(name any, alias string) query.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.Table = expr.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm UpdateQM) NotIndexed() query.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		var s string
		q.IndexedBy = &s
	})
}

func (qm UpdateQM) IndexedBy(indexName string) query.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.IndexedBy = &indexName
	})
}

func (qm UpdateQM) Set(a, b any) query.Mod[*UpdateQuery] {
	return mods.Set[*UpdateQuery]{builder.OP("=", a, b)}
}

func (qm UpdateQM) SetArg(a, b any) query.Mod[*UpdateQuery] {
	return mods.Set[*UpdateQuery]{builder.OP("=", a, qm.Arg(b))}
}

func (qm UpdateQM) Where(e query.Expression) query.Mod[*UpdateQuery] {
	return mods.Where[*UpdateQuery]{e}
}

func (qm UpdateQM) WhereClause(clause string, args ...any) query.Mod[*UpdateQuery] {
	return mods.Where[*UpdateQuery]{qm.Raw(clause, args...)}
}

func (qm UpdateQM) Returning(expressions ...any) query.Mod[*UpdateQuery] {
	return mods.Returning[*UpdateQuery](expressions)
}
