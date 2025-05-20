package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/delete.html
type DeleteQuery struct {
	hints

	clause.With
	modifiers[string]
	Tables []clause.TableRef

	// This is needed since Paritions come AFTER the table alias in DELETE statements
	// In other statements, they come before the table alias
	Partitions []string

	clause.TableRef
	clause.Where
	clause.OrderBy
	clause.Limit
	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*DeleteQuery]
}

func (d *DeleteQuery) AppendTable(expr bob.Expression) {
	d.Tables = append(d.Tables, clause.TableRef{
		Expression: expr,
	})
}

func (d *DeleteQuery) AppendPartition(partitions ...string) {
	d.Partitions = append(d.Partitions, partitions...)
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

	w.Write([]byte("DELETE "))

	// no optimizer hint args
	_, err = bob.ExpressIf(ctx, w, dl, start+len(args), d.hints,
		len(d.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(ctx, w, dl, start+len(args), d.modifiers,
		len(d.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	tableArgs, err := bob.ExpressSlice(ctx, w, dl, start+len(args), d.Tables, "FROM ", ", ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	_, err = bob.ExpressSlice(ctx, w, dl, start, d.Partitions, " PARTITION (", ", ", ")")
	if err != nil {
		return nil, err
	}

	usingArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.TableRef,
		d.TableRef.Expression != nil, "\nUSING ", "")
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

	orderArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.OrderBy,
		len(d.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = bob.ExpressIf(ctx, w, dl, start+len(args), d.Limit,
		d.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
