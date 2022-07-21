package psql

import (
	"io"

	"github.com/jinzhu/copier"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Insert(mods ...mods.QueryMod[*InsertQuery]) *InsertQuery {
	s := &InsertQuery{}
	for _, mod := range mods {
		mod.Apply(s)
	}

	return s
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

func (i *InsertQuery) Clone() *InsertQuery {
	var i2 = new(InsertQuery)
	copier.CopyWithOption(i2, i, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
	})

	return i2
}

func (i *InsertQuery) Apply(mods ...mods.QueryMod[*InsertQuery]) {
	for _, mod := range mods {
		mod.Apply(i)
	}
}

func (i InsertQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return i.WriteSQL(w, dialect, start)
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
	withMod[*InsertQuery]
}

func (qm InsertQM) Into(name any, columns ...string) mods.QueryMod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = expr.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) IntoAs(name any, alias string, columns ...string) mods.QueryMod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = expr.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func (qm InsertQM) OverridingSystem() mods.QueryMod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.overriding = "SYSTEM"
	})
}

func (qm InsertQM) OverridingUser() mods.QueryMod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.overriding = "USER"
	})
}

func (qm InsertQM) Values(expressions ...any) mods.QueryMod[*InsertQuery] {
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

func (qm InsertQM) Returning(expressions ...any) mods.QueryMod[*InsertQuery] {
	return mods.Returning[*InsertQuery](expressions)
}
