package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Update(queryMods ...query.Mod[*updateQuery]) query.BaseQuery[*updateQuery] {
	q := &updateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*updateQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_update.html
type updateQuery struct {
	clause.With
	or
	clause.FromItem
	clause.Set
	clause.FromItems
	clause.Where
	clause.Returning
}

func (u updateQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
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
	withMod[*updateQuery]      // For CTEs
	mods.FromMod[*updateQuery] // update *FROM*
	joinMod[*clause.FromItem]  // joins, which are mods of the FROM
	fromItemMod                // Dialect specific fromItem mods
	orMod[*updateQuery]        // UPDATE or REPLACE|ABORT|IGNORE e.t.c.
}

func (qm UpdateQM) Table(name any) query.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm UpdateQM) TableAs(name any, alias string) query.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm UpdateQM) NotIndexed() query.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		var s string
		q.IndexedBy = &s
	})
}

func (qm UpdateQM) IndexedBy(indexName string) query.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.IndexedBy = &indexName
	})
}

func (qm UpdateQM) Set(a string, b any) query.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), b)}
}

func (qm UpdateQM) SetArg(a string, b any) query.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), Arg(b))}
}

func (qm UpdateQM) Where(e query.Expression) query.Mod[*updateQuery] {
	return mods.Where[*updateQuery]{e}
}

func (qm UpdateQM) WhereClause(clause string, args ...any) query.Mod[*updateQuery] {
	return mods.Where[*updateQuery]{Raw(clause, args...)}
}

func (qm UpdateQM) Returning(clauses ...any) query.Mod[*updateQuery] {
	return mods.Returning[*updateQuery](clauses)
}
