package psql

import (
	"io"

	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/mods"
	"github.com/stephenafamo/typesql/query"
)

func Insert(mods ...mods.QueryMod[*InsertQuery]) *InsertQuery {
	s := &InsertQuery{}
	for _, mod := range mods {
		mod.Apply(s)
	}

	return s
}

// Not handling on-conflict yet
type InsertQuery struct {
	expr.With
	overriding string
	expr.Table
	expr.Values
	expr.Conflict
	expr.Returning
}

func (i InsertQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return i.WriteSQL(w, dialect, start)
}

func (i InsertQuery) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
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

type InsertQM struct{}

func (qm InsertQM) With(name string, columns ...string) cteChain[*InsertQuery] {
	return cteChain[*InsertQuery](func() expr.CTE {
		return expr.CTE{
			Name:    name,
			Columns: columns,
		}
	})
}

func (qm InsertQM) Recursive(r bool) mods.QueryMod[*InsertQuery] {
	return mods.Recursive[*InsertQuery](r)
}

func (qm InsertQM) Into(name any, columns ...string) mods.QueryMod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Table = expr.Table{
			Expression: name,
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
func (qm InsertQM) OnConflict(column any, where ...any) onConflict[*InsertQuery] {
	return onConflict[*InsertQuery](func() expr.Conflict {
		return expr.Conflict{
			Target: expr.ConflictTarget{
				Target: expr.Group(column),
				Where:  where,
			},
		}
	})
}

func (qm InsertQM) Returning(expressions ...any) mods.QueryMod[*InsertQuery] {
	return mods.Returning[*InsertQuery](expressions)
}
