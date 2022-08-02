package psql

import (
	"io"

	"github.com/stephenafamo/bob"
	clause "github.com/stephenafamo/bob/clause"
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
// https://www.postgresql.org/docs/current/sql-update.html
type updateQuery struct {
	clause.With
	only bool
	clause.Table
	clause.Set
	clause.FromItems
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

	fromArgs, err := bob.ExpressSlice(w, d, start+len(args), u.FromItems.Items, "\nFROM ", ",\n", "")
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
	withMod[*updateQuery]
	mods.FromMod[*updateQuery]
	fromItemMod
	joinMod[*clause.FromItem]
}

func (qm UpdateQM) Only() bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(u *updateQuery) {
		u.only = true
	})
}

func (qm UpdateQM) Table(name any) bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(u *updateQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm UpdateQM) TableAs(name any, alias string) bob.Mod[*updateQuery] {
	return mods.QueryModFunc[*updateQuery](func(u *updateQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm UpdateQM) Set(a string, b any) bob.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), b)}
}

func (qm UpdateQM) SetArg(a string, b any) bob.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), Arg(b))}
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
