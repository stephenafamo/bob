package sqlite

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
// https://www.sqlite.org/lang_delete.html
type deleteQuery struct {
	clause.With
	clause.FromItem
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
	withMod[*deleteQuery]
}

func (qm DeleteQM) From(name any) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		q.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		q.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) NotIndexed() query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		var s string
		q.IndexedBy = &s
	})
}

func (qm DeleteQM) IndexedBy(indexName string) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		q.IndexedBy = &indexName
	})
}

func (qm DeleteQM) Where(e query.Expression) query.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) query.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{Raw(clause, args...)}
}

func (qm DeleteQM) Returning(clauseessions ...any) query.Mod[*deleteQuery] {
	return mods.Returning[*deleteQuery](clauseessions)
}
