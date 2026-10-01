package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the insert query structure as documented in
// https://learn.microsoft.com/en-us/sql/t-sql/statements/insert-transact-sql
type InsertQuery struct {
	clause.With
	Top *Top
	clause.TableRef
	Output
	clause.Values

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*InsertQuery]
}

func (i InsertQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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

	w.WriteString("INSERT ")

	topArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Top,
		i.Top != nil, "", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, topArgs...)

	tableArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.TableRef,
		true, "INTO ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	// OUTPUT comes between table and VALUES in T-SQL
	outputArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Output,
		i.Output.HasOutput(), "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, outputArgs...)

	valArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Values, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	w.WriteString("\n")
	return args, nil
}
