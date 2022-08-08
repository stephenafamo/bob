package psql

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
// https://www.postgresql.org/docs/current/sql-insert.html
type InsertQuery struct {
	clause.With
	overriding string
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

	tableArgs, err := bob.ExpressIf(w, d, start+len(args), i.Table,
		true, "INSERT INTO ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	_, err = bob.ExpressIf(w, d, start+len(args), i.overriding,
		i.overriding != "", "\nOVERRIDING ", " VALUE")
	if err != nil {
		return nil, err
	}

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
	withMod[*InsertQuery]
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

func (qm insertQM) OverridingSystem() bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.overriding = "SYSTEM"
	})
}

func (qm insertQM) OverridingUser() bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.overriding = "USER"
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

// The column to target. Will auto add brackets
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

func (qm insertQM) OnConflictOnConstraint(constraint string) mods.Conflict[*InsertQuery] {
	return mods.Conflict[*InsertQuery](func() clause.Conflict {
		return clause.Conflict{
			Target: clause.ConflictTarget{
				Target: `ON CONSTRAINT "` + constraint + `"`,
			},
		}
	})
}

func (qm insertQM) Returning(clauses ...any) bob.Mod[*InsertQuery] {
	return mods.Returning[*InsertQuery](clauses)
}
