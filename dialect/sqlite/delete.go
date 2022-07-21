package sqlite

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
// https://www.sqlite.org/lang_delete.html
type DeleteQuery struct {
	expr.With
	expr.FromItem
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

	w.Write([]byte("DELETE FROM"))

	tableArgs, err := query.ExpressIf(w, dl, start+len(args), d.FromItem, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

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
}

func (qm DeleteQM) From(name any) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.Table = expr.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.Table = expr.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) NotIndexed() query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		var s string
		q.IndexedBy = &s
	})
}

func (qm DeleteQM) IndexedBy(indexName string) query.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.IndexedBy = &indexName
	})
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
