package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-delete.html
type DeleteQuery struct {
	clause.With
	Only bool
	clause.Table
	clause.From
	clause.Where
	clause.Returning
	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*DeleteQuery]
}

func (d DeleteQuery) WriteSQL(ctx context.Context, w io.Writer, dl bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	if ctx, err = d.RunContextualMods(ctx, &d); err != nil {
		return nil, err
	}

	withArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.With,
		len(d.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("DELETE FROM "))

	if d.Only {
		w.Write([]byte("ONLY "))
	}

	tableArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.From,
		d.From.Table != nil, "\nUSING ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	whereArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Where,
		len(d.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Returning,
		len(d.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}
