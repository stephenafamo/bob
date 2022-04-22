package psql

import (
	"io"

	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/mods"
	"github.com/stephenafamo/typesql/query"
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

func (u DeleteQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return u.WriteSQL(w, dialect, start)
}

func (u DeleteQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	tableArgs, err := query.ExpressIf(w, d, start+len(args), u.Table, true, "DELETE FROM ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := query.ExpressSlice(w, d, start+len(args), u.FromItems.Items, "\nUSING ", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	whereArgs, err := query.ExpressIf(w, d, start+len(args), u.Where,
		len(u.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := query.ExpressIf(w, d, start+len(args), u.Returning,
		len(u.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}

type DeleteQM struct {
	joinMod[*expr.FromItem]
}

func (qm DeleteQM) With(name string, columns ...string) cteChain[*DeleteQuery] {
	return cteChain[*DeleteQuery](func() expr.CTE {
		return expr.CTE{
			Name:    name,
			Columns: columns,
		}
	})
}

func (qm DeleteQM) Recursive(r bool) mods.QueryMod[*DeleteQuery] {
	return mods.Recursive[*DeleteQuery](r)
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
	f := &expr.FromItem{
		Table: table,
	}
	for _, mod := range usingMods {
		mod.Apply(f)
	}

	return mods.FromItems[*DeleteQuery](*f)
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
