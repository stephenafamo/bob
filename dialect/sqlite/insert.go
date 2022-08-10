package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Insert(queryMods ...bob.Mod[*InsertQuery]) bob.BaseQuery[*InsertQuery] {
	q := &InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*InsertQuery]{
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

func (i InsertQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, d, start+len(args), i.With,
		len(i.With.CTEs) > 0, "", "\n")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("INSERT"))

	_, err = bob.ExpressIf(w, d, start+len(args), i.or, true, " ", "")
	if err != nil {
		return nil, err
	}

	tableArgs, err := bob.ExpressIf(w, d, start+len(args), i.Table, true, " INTO ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	valArgs, err := bob.ExpressIf(w, d, start+len(args), i.Values, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	retArgs, err := bob.ExpressIf(w, d, start+len(args), i.Returning,
		len(i.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	conflictArgs, err := bob.ExpressIf(w, d, start+len(args), i.Conflict,
		i.Conflict.Do != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, conflictArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

//nolint:gochecknoglobals
var InsertQM = insertQM{}

type insertQM struct {
	withMod[*InsertQuery] // For CTEs
	orMod[*InsertQuery]   // INSERT or REPLACE|ABORT|IGNORE e.t.c.
}

func (qm insertQM) Into(name any, columns ...string) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func (qm insertQM) IntoAs(name any, alias string, columns ...string) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func (qm insertQM) Values(clauses ...any) bob.Mod[*InsertQuery] {
	return mods.Values[*InsertQuery](clauses)
}

// Insert from a query
// If Go allows type parameters on methods, limit this to select and raw
func (qm insertQM) Query(q bob.Query) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Query = q
	})
}

func (qm insertQM) OnConflict(column any, where ...any) mods.Conflict[*InsertQuery] {
	if column != nil {
		column = P(column)
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

func (qm insertQM) Returning(clauses ...any) bob.Mod[*InsertQuery] {
	return mods.Returning[*InsertQuery](clauses)
}
