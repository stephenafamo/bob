package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func Update(queryMods ...bob.Mod[*updateQuery]) bob.BaseQuery[*updateQuery] {
	q := &updateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*updateQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_update.html
type updateQuery struct {
	clause.With
	or
	table clause.From
	clause.Set
	clause.From
	clause.Where
	clause.Returning
}

func (u updateQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
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

type UpdateQM struct {
	withMod[*updateQuery]     // For CTEs
	joinMod[*clause.From]     // joins, which are mods of the FROM
	fromItemMod[*updateQuery] // Dialect specific fromItem mods
	orMod[*updateQuery]       // UPDATE or REPLACE|ABORT|IGNORE e.t.c.
}

func (qm UpdateQM) Table(name any) bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.table.Table = name
	})
}

func (qm UpdateQM) TableAs(name any, alias string) bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.table.Table = name
		q.table.Alias = alias
	})
}

func (qm UpdateQM) TableIndexedBy(i string) bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.table.IndexedBy = &i
	})
}

func (qm UpdateQM) TableNotIndexed() bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		var s string
		q.table.IndexedBy = &s
	})
}

func (qm UpdateQM) Set(a string, b any) bob.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), b)}
}

func (qm UpdateQM) SetArg(a string, b any) bob.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), Arg(b))}
}

func (UpdateQM) From(table any) bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(q *updateQuery) {
		q.SetTable(table)
	})
}

func (qm UpdateQM) Where(e bob.Expression) bob.Mod[*updateQuery] {
	return mods.Where[*updateQuery]{e}
}

func (qm UpdateQM) WhereClause(clause string, args ...any) bob.Mod[*updateQuery] {
	return mods.Where[*updateQuery]{Raw(clause, args...)}
}

func (qm UpdateQM) Returning(clauses ...any) bob.Mod[*updateQuery] {
	return mods.Returning[*updateQuery](clauses)
}
