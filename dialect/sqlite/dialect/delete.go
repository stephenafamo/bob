package dialect

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

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
