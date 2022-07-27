package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Insert(queryMods ...query.Mod[*InsertQuery]) query.BaseQuery[*InsertQuery] {
	q := &InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*InsertQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_insert.html
type InsertQuery struct {
	clause.With
	or
	clause.Table
	clause.Values
	clause.Conflict
	clause.Returning
}

func (i InsertQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), i.With,
		len(i.With.CTEs) > 0, "", "\n")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("INSERT"))

	_, err = query.ExpressIf(w, d, start+len(args), i.or, true, " ", "")
	if err != nil {
		return nil, err
	}

	tableArgs, err := query.ExpressIf(w, d, start+len(args), i.Table, true, " INTO ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	valArgs, err := query.ExpressIf(w, d, start+len(args), i.Values, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	retArgs, err := query.ExpressIf(w, d, start+len(args), i.Returning,
		len(i.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	conflictArgs, err := query.ExpressIf(w, d, start+len(args), i.Conflict,
		i.Conflict.Do != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, conflictArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

type InsertQM struct {
	builderMod
	withMod[*InsertQuery] // For CTEs
	orMod[*InsertQuery]   // INSERT or REPLACE|ABORT|IGNORE e.t.c.
}

func (qm InsertQM) Into(name any, columns ...string) query.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) IntoAs(name any, alias string, columns ...string) query.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) Values(clauseessions ...any) query.Mod[*InsertQuery] {
	return mods.Values[*InsertQuery](clauseessions)
}

func (qm InsertQM) OnConflict(column any, where ...any) mods.Conflict[*InsertQuery] {
	if column != nil {
		column = qm.P(column)
	}
	return mods.Conflict[*InsertQuery](func() clause.Conflict {
		return clause.Conflict{
			Target: clause.ConflictTarget{
				Target: column,
				Where:  where,
			},
		}
	})
}

func (qm InsertQM) Returning(clauseessions ...any) query.Mod[*InsertQuery] {
	return mods.Returning[*InsertQuery](clauseessions)
}
