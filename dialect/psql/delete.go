package psql

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Delete(queryMods ...query.Mod[*deleteQuery]) query.BaseQuery[*deleteQuery] {
	q := &deleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*deleteQuery]{
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

func (d deleteQuery) WriteSQL(w io.Writer, dl query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, dl, start+len(args), d.With,
		len(d.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("DELETE FROM "))

	if d.only {
		w.Write([]byte("ONLY "))
	}

	tableArgs, err := query.ExpressIf(w, dl, start+len(args), d.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := query.ExpressSlice(w, dl, start+len(args), d.FromItems.Items, "\nUSING ", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	whereArgs, err := query.ExpressIf(w, dl, start+len(args), d.Where,
		len(d.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := query.ExpressIf(w, dl, start+len(args), d.Returning,
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

func (qm DeleteQM) Only() query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(d *deleteQuery) {
		d.only = true
	})
}

func (qm DeleteQM) From(name any) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(u *deleteQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(u *deleteQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) Using(table any, usingMods ...query.Mod[*clause.FromItem]) query.Mod[*deleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm DeleteQM) Where(e query.Expression) query.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) query.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{Raw(clause, args...)}
}

func (qm DeleteQM) Returning(clauses ...any) query.Mod[*deleteQuery] {
	return mods.Returning[*deleteQuery](clauses)
}
