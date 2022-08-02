package psql

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Delete(queryMods ...bob.Mod[*deleteQuery]) bob.BaseQuery[*deleteQuery] {
	q := &deleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*deleteQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-delete.html
type deleteQuery struct {
	clause.With
	only bool
	clause.Table
	clause.FromItems
	clause.Where
	clause.Returning
}

func (d deleteQuery) WriteSQL(w io.Writer, dl bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, dl, start+len(args), d.With,
		len(d.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("DELETE FROM "))

	if d.only {
		w.Write([]byte("ONLY "))
	}

	tableArgs, err := bob.ExpressIf(w, dl, start+len(args), d.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := bob.ExpressSlice(w, dl, start+len(args), d.FromItems.Items, "\nUSING ", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	whereArgs, err := bob.ExpressIf(w, dl, start+len(args), d.Where,
		len(d.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := bob.ExpressIf(w, dl, start+len(args), d.Returning,
		len(d.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}

type DeleteQM struct {
	withMod[*deleteQuery]
	mods.FromMod[*deleteQuery]
	fromItemMod
	joinMod[*clause.FromItem]
}

func (qm DeleteQM) Only() bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(d *deleteQuery) {
		d.only = true
	})
}

func (qm DeleteQM) From(name any) bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(u *deleteQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(u *deleteQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) Using(table any, usingMods ...bob.Mod[*clause.FromItem]) bob.Mod[*deleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm DeleteQM) Where(e bob.Expression) bob.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) bob.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{Raw(clause, args...)}
}

func (qm DeleteQM) Returning(clauses ...any) bob.Mod[*deleteQuery] {
	return mods.Returning[*deleteQuery](clauses)
}
