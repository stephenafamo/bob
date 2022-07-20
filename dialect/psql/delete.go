package psql

import (
	"io"

	"github.com/jinzhu/copier"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Delete(mods ...mods.QueryMod[*DeleteQuery]) *DeleteQuery {
	s := &DeleteQuery{}
	for _, mod := range mods {
		mod.Apply(s)
	}

	return s
}

// Not handling on-conflict yet
type DeleteQuery struct {
	expr.With
	expr.Table
	expr.FromItems
	expr.Where
	expr.Returning
}

func (d *DeleteQuery) Clone() *DeleteQuery {
	var d2 = new(DeleteQuery)
	copier.CopyWithOption(d2, d, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
	})

	return d2
}

func (d *DeleteQuery) Apply(mods ...mods.QueryMod[*DeleteQuery]) {
	for _, mod := range mods {
		mod.Apply(d)
	}
}

func (d DeleteQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return d.WriteSQL(w, dialect, start)
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

func (qm DeleteQM) From(name any) mods.QueryMod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = expr.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) mods.QueryMod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.Table = expr.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) Using(table any, usingMods ...mods.QueryMod[*expr.FromItem]) mods.QueryMod[*DeleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm DeleteQM) Where(e query.Expression) mods.QueryMod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) mods.QueryMod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{expr.Statement(clause, args...)}
}

func (qm DeleteQM) Returning(expressions ...any) mods.QueryMod[*DeleteQuery] {
	return mods.Returning[*DeleteQuery](expressions)
}
