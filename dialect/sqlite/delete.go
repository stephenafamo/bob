package sqlite

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
// https://www.sqlite.org/lang_delete.html
type deleteQuery struct {
	clause.With
	clause.FromItem
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

	w.Write([]byte("DELETE FROM"))

	tableArgs, err := bob.ExpressIf(w, dl, start+len(args), d.FromItem, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

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
}

func (qm DeleteQM) From(name any) bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		q.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm DeleteQM) FromAs(name any, alias string) bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		q.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm DeleteQM) NotIndexed() bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		var s string
		q.IndexedBy = &s
	})
}

func (qm DeleteQM) IndexedBy(indexName string) bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(q *deleteQuery) {
		q.IndexedBy = &indexName
	})
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
