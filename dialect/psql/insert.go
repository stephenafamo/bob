package psql

import (
	"io"

	"github.com/stephenafamo/bob/expr"
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
		Dialect:    Dialect{},
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-insert.html
type InsertQuery struct {
	expr.With
	overriding string
	expr.Table
	expr.Values
	expr.Conflict
	expr.Returning
}

func (i InsertQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
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
	BuilderMod
	withMod[*InsertQuery]
}

func (qm InsertQM) Into(name any, columns ...string) query.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = expr.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) IntoAs(name any, alias string, columns ...string) query.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = expr.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) OverridingSystem() query.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.overriding = "SYSTEM"
	})
}

func (qm InsertQM) OverridingUser() query.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.overriding = "USER"
	})
}

func (qm InsertQM) Values(expressions ...any) query.Mod[*InsertQuery] {
	return mods.Values[*InsertQuery](expressions)
}

// The column to target. Will auto add brackets
func (qm InsertQM) OnConflict(column any, where ...any) mods.Conflict[*InsertQuery] {
	if column != nil {
		column = expr.P(column)
	}
	return mods.Conflict[*InsertQuery](func() expr.Conflict {
		return expr.Conflict{
			Target: expr.ConflictTarget{
				Target: column,
				Where:  where,
			},
		}
	})
}

func (qm InsertQM) OnConflictOnConstraint(constraint string) mods.Conflict[*InsertQuery] {
	return mods.Conflict[*InsertQuery](func() expr.Conflict {
		return expr.Conflict{
			Target: expr.ConflictTarget{
				Target: `ON CONSTRAINT "` + constraint + `"`,
			},
		}
	})
}

func (qm InsertQM) Returning(expressions ...any) query.Mod[*InsertQuery] {
	return mods.Returning[*InsertQuery](expressions)
}
