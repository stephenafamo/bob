package psql

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Insert(queryMods ...query.Mod[*insertQuery]) query.BaseQuery[*insertQuery] {
	q := &insertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*insertQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-insert.html
type insertQuery struct {
	clause.With
	overriding string
	clause.Table
	clause.Values
	clause.Conflict
	clause.Returning
}

func (i insertQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), i.With,
		len(i.With.CTEs) > 0, "", "\n")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	tableArgs, err := query.ExpressIf(w, d, start+len(args), i.Table,
		true, "INSERT INTO ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	_, err = query.ExpressIf(w, d, start+len(args), i.overriding,
		i.overriding != "", "\nOVERRIDING ", " VALUE")
	if err != nil {
		return nil, err
	}

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
	withMod[*insertQuery]
}

func (qm InsertQM) Into(name any, columns ...string) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) IntoAs(name any, alias string, columns ...string) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) OverridingSystem() query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.overriding = "SYSTEM"
	})
}

func (qm InsertQM) OverridingUser() query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.overriding = "USER"
	})
}

func (qm InsertQM) Values(clauses ...any) query.Mod[*insertQuery] {
	return mods.Values[*insertQuery](clauses)
}

// Insert from a query
// If Go allows type parameters on methods, limit this to select and raw
func (qm InsertQM) Query(q query.Query) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.Query = q
	})
}

// The column to target. Will auto add brackets
func (qm InsertQM) OnConflict(column any, where ...any) mods.Conflict[*insertQuery] {
	if column != nil {
		column = P(column)
	}
	return mods.Conflict[*insertQuery](func() clause.Conflict {
		return clause.Conflict{
			Target: clause.ConflictTarget{
				Target: column,
				Where:  where,
			},
		}
	})
}

func (qm InsertQM) OnConflictOnConstraint(constraint string) mods.Conflict[*insertQuery] {
	return mods.Conflict[*insertQuery](func() clause.Conflict {
		return clause.Conflict{
			Target: clause.ConflictTarget{
				Target: `ON CONSTRAINT "` + constraint + `"`,
			},
		}
	})
}

func (qm InsertQM) Returning(clauses ...any) query.Mod[*insertQuery] {
	return mods.Returning[*insertQuery](clauses)
}
