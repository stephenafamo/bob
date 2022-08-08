package psql

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Delete(queryMods ...bob.Mod[*DeleteQuery]) bob.BaseQuery[*DeleteQuery] {
	q := &DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*DeleteQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-delete.html
type DeleteQuery struct {
	clause.With
	only bool
	clause.Table
	clause.FromItems
	clause.Where
	clause.Returning
}

func (d DeleteQuery) WriteSQL(w io.Writer, dl bob.Dialect, start int) ([]any, error) {
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

//nolint:gochecknoglobals
var DeleteQM = deleteQM{}

type deleteQM struct {
	withMod[*DeleteQuery]
	mods.FromMod[*DeleteQuery]
	fromItemMod
	joinMod[*clause.FromItem]
}

func (qm deleteQM) Only() bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(d *DeleteQuery) {
		d.only = true
	})
}

func (qm deleteQM) From(name any) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm deleteQM) FromAs(name any, alias string) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm deleteQM) Using(table any, usingMods ...bob.Mod[*clause.FromItem]) bob.Mod[*DeleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm deleteQM) Where(e bob.Expression) bob.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{e}
}

func (qm deleteQM) WhereClause(clause string, args ...any) bob.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{Raw(clause, args...)}
}

func (qm deleteQM) Returning(clauses ...any) bob.Mod[*DeleteQuery] {
	return mods.Returning[*DeleteQuery](clauses)
}
