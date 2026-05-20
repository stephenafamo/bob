package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// DeleteQuery tries to represent the DELETE query structure as documented in
// https://www.postgresql.org/docs/current/sql-delete.html
type DeleteQuery struct {
	clause.With
	Only  bool
	Table clause.TableRef
	clause.TableRef
	UsingItems []clause.TableRef
	clause.WhereCurrentOf
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*DeleteQuery]
}

func (d *DeleteQuery) SetTable(table any) {
	d.TableRef.SetTable(table)
	d.UsingItems = nil
}

func (d *DeleteQuery) AppendTableRef(using clause.TableRef) {
	d.UsingItems = append(d.UsingItems, using)
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

	w.WriteString("DELETE FROM ")

	if d.Only {
		w.WriteString("ONLY ")
	}

	tableArgs, err := bob.ExpressIf(ctx, w, dl, start+len(args), d.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	args, err = writeDeleteUsing(ctx, w, dl, start+len(args), args, d.TableRef, d.UsingItems)
	if err != nil {
		return nil, err
	}

	whereArgs, err := clause.WriteWhereAndCurrentOf(ctx, w, dl, start+len(args), d.Where, d.WhereCurrentOf)
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

func writeDeleteUsing(
	ctx context.Context, w io.StringWriter, dl bob.Dialect, start int,
	args []any, tableRef clause.TableRef, usingItems []clause.TableRef,
) ([]any, error) {
	if tableRef.Expression == nil && len(usingItems) == 0 {
		return args, nil
	}
	w.WriteString("\nUSING ")

	if tableRef.Expression != nil {
		usingArgs, err := bob.Express(ctx, w, dl, start, tableRef)
		if err != nil {
			return nil, err
		}
		args = append(args, usingArgs...)
	}

	if len(usingItems) > 0 {
		prefix := ""
		if tableRef.Expression != nil {
			prefix = ", "
		}

		itemArgs, err := bob.ExpressSlice(ctx, w, dl, start+len(args), usingItems, prefix, ", ", "")
		if err != nil {
			return nil, err
		}
		args = append(args, itemArgs...)
	}
	return args, nil
}
