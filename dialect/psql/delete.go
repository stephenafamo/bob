package psql

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Delete(queryMods ...query.Mod[*DeleteQuery]) query.BaseQuery[*DeleteQuery] {
	q := &DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*DeleteQuery]{
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

func (d DeleteQuery) WriteSQL(w io.Writer, dl query.Dialect, start int) ([]any, error) {
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
	BuilderMod
	withMod[*DeleteQuery]
	mods.FromMod[*DeleteQuery]
	fromItemMod
	joinMod[*clause.FromItem]
}

func (qm DeleteQM) Only() query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(d *DeleteQuery) {
		d.only = true
	})
}

func (qm DeleteQM) From(name any) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) Using(table any, usingMods ...query.Mod[*clause.FromItem]) query.Mod[*DeleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm DeleteQM) Where(e query.Expression) query.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) query.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{qm.Raw(clause, args...)}
}

func (qm DeleteQM) Returning(clauseessions ...any) query.Mod[*DeleteQuery] {
	return mods.Returning[*DeleteQuery](clauseessions)
}
