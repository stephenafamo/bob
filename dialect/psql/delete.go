package psql

import (
	"io"

	"github.com/stephenafamo/bob/expr"
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
		Dialect:    Dialect{},
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-delete.html
type DeleteQuery struct {
	expr.With
	expr.Table
	expr.FromItems
	expr.Where
	expr.Returning
}

func (d DeleteQuery) WriteSQL(w io.Writer, dl query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, dl, start+len(args), d.With,
		len(d.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	tableArgs, err := query.ExpressIf(w, dl, start+len(args), d.Table, true, "DELETE FROM ", "")
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
	withMod[*DeleteQuery]
	mods.FromMod[*DeleteQuery]
	fromItemMod
	joinMod[*expr.FromItem]
}

func (qm DeleteQM) From(name any) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = expr.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = expr.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) Using(table any, usingMods ...query.Mod[*expr.FromItem]) query.Mod[*DeleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm DeleteQM) Where(e query.Expression) query.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) query.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{expr.Statement(clause, args...)}
}

func (qm DeleteQM) Returning(expressions ...any) query.Mod[*DeleteQuery] {
	return mods.Returning[*DeleteQuery](expressions)
}
