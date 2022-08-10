package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func Update(queryMods ...bob.Mod[*UpdateQuery]) bob.BaseQuery[*UpdateQuery] {
	q := &UpdateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*UpdateQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_update.html
type UpdateQuery struct {
	clause.With
	or
	table clause.From
	clause.Set
	clause.From
	clause.Where
	clause.Returning
}

func (u UpdateQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("UPDATE"))

	_, err = bob.ExpressIf(w, d, start+len(args), u.or, true, " ", "")
	if err != nil {
		return nil, err
	}

	tableArgs, err := bob.ExpressIf(w, d, start+len(args), u.table, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	setArgs, err := bob.ExpressIf(w, d, start+len(args), u.Set, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	fromArgs, err := bob.ExpressIf(w, d, start+len(args), u.From,
		u.From.Table != nil, "\nFROM ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := bob.ExpressIf(w, d, start+len(args), u.Where,
		len(u.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := bob.ExpressIf(w, d, start+len(args), u.Returning,
		len(u.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}

//nolint:gochecknoglobals
var UpdateQM = updateQM{}

type updateQM struct {
	withMod[*UpdateQuery]     // For CTEs
	joinMod[*clause.From]     // joins, which are mods of the FROM
	fromItemMod[*UpdateQuery] // Dialect specific fromItem mods
	orMod[*UpdateQuery]       // UPDATE or REPLACE|ABORT|IGNORE e.t.c.
}

func (qm updateQM) Table(name any) bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.table.Table = name
	})
}

func (qm updateQM) TableAs(name any, alias string) bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.table.Table = name
		q.table.Alias = alias
	})
}

func (qm updateQM) TableIndexedBy(i string) bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.table.IndexedBy = &i
	})
}

func (qm updateQM) TableNotIndexed() bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		var s string
		q.table.IndexedBy = &s
	})
}

func (qm updateQM) Set(a string, b any) bob.Mod[*UpdateQuery] {
	return mods.Set[*UpdateQuery]{expr.OP("=", Quote(a), b)}
}

func (qm updateQM) SetArg(a string, b any) bob.Mod[*UpdateQuery] {
	return mods.Set[*UpdateQuery]{expr.OP("=", Quote(a), Arg(b))}
}

func (updateQM) From(table any) bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(q *UpdateQuery) {
		q.SetTable(table)
	})
}

func (qm updateQM) Where(e bob.Expression) bob.Mod[*UpdateQuery] {
	return mods.Where[*UpdateQuery]{e}
}

func (qm updateQM) WhereClause(clause string, args ...any) bob.Mod[*UpdateQuery] {
	return mods.Where[*UpdateQuery]{Raw(clause, args...)}
}

func (qm updateQM) Returning(clauses ...any) bob.Mod[*UpdateQuery] {
	return mods.Returning[*UpdateQuery](clauses)
}
