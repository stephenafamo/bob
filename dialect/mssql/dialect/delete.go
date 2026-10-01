package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the delete query structure as documented in
// https://learn.microsoft.com/en-us/sql/t-sql/statements/delete-transact-sql
type DeleteQuery struct {
	clause.With
	Top   *Top
	Table clause.TableRef
	Output
	clause.TableRef
	clause.Where

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*DeleteQuery]
}

func (d DeleteQuery) WriteSQL(ctx context.Context, w io.StringWriter, dl bob.Dialect, start int) ([]any, error) {
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

	w.WriteString("DELETE ")

	topArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Top,
		d.Top != nil, "", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, topArgs...)

	w.WriteString("FROM ")

	tableArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	// OUTPUT comes between FROM table and WHERE in T-SQL
	outputArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Output,
		d.Output.HasOutput(), "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, outputArgs...)

	fromArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.TableRef,
		d.TableRef.Expression != nil, "\nFROM ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Where,
		len(d.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	return args, nil
}
