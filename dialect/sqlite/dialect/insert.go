package dialect

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

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

	conflictArgs, err := bob.ExpressIf(w, d, start+len(args), i.Conflict,
		i.Conflict.Do != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, conflictArgs...)

	retArgs, err := bob.ExpressIf(w, d, start+len(args), i.Returning,
		len(i.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	w.Write([]byte("\n"))
	return args, nil
}
