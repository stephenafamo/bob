package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-insert.html
type InsertQuery struct {
	clause.With
	Overriding string
	clause.TableRef
	clause.Values
	clause.Conflict
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*InsertQuery]
}

func (i InsertQuery) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	if ctx, err = i.RunContextualMods(ctx, &i); err != nil {
		return nil, err
	}

	withArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.With,
		len(i.With.CTEs) > 0, "", "\n")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	tableArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.TableRef,
		true, "INSERT INTO ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	_, err = bob.ExpressIf(ctx, w, d, start+len(args), i.Overriding,
		i.Overriding != "", "\nOVERRIDING ", " VALUE")
	if err != nil {
		return nil, err
	}

	valArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Values, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	conflictArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Conflict.Expression,
		i.Conflict.Expression != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, conflictArgs...)

	retArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Returning,
		len(i.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	w.Write([]byte("\n"))
	return args, nil
}
