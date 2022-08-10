package sqlite

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
// https://www.sqlite.org/lang_delete.html
type DeleteQuery struct {
	clause.With
	clause.From
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

	w.Write([]byte("DELETE FROM"))

	tableArgs, err := bob.ExpressIf(w, dl, start+len(args), d.From, true, " ", "")
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

//nolint:gochecknoglobals
var DeleteQM = deleteQM{}

type deleteQM struct {
	withMod[*DeleteQuery]
}

func (qm deleteQM) From(name any) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.Table = clause.Table{
			Expression: name,
		}
	})
}

func (qm deleteQM) FromAs(name any, alias string) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.Table = clause.Table{
			Expression: name,
			Alias:      alias,
		}
	})
}

func (qm deleteQM) NotIndexed() bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		var s string
		q.IndexedBy = &s
	})
}

func (qm deleteQM) IndexedBy(indexName string) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.IndexedBy = &indexName
	})
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
