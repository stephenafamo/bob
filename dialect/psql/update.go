package psql

import (
	"io"

	"github.com/stephenafamo/bob"
	clause "github.com/stephenafamo/bob/clause"
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
// https://www.postgresql.org/docs/current/sql-update.html
type UpdateQuery struct {
	clause.With
	only bool
	clause.Table
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

	w.Write([]byte("UPDATE "))

	if u.only {
		w.Write([]byte("ONLY "))
	}

	tableArgs, err := bob.ExpressIf(w, d, start+len(args), u.Table, true, "", "")
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
	withMod[*UpdateQuery]
	fromItemMod[*UpdateQuery]
	joinMod[*clause.From]
}

func (qm updateQM) Only() bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(u *UpdateQuery) {
		u.only = true
	})
}

func (qm updateQM) Table(name any) bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(u *UpdateQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm updateQM) TableAs(name any, alias string) bob.Mod[*UpdateQuery] {
	return mods.QueryModFunc[*UpdateQuery](func(u *UpdateQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
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
