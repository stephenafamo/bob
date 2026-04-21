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
	Only  bool
	Table clause.TableRef
	clause.TableRef
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*DeleteQuery]
}

func (d *DeleteQuery) SetTargetOnly(only bool) {
	d.Table.SetOnly(only)
}

func (d *DeleteQuery) SetTargetTable(table any) {
	d.Table.SetTable(table)
}

func (d *DeleteQuery) SetTargetTableAlias(alias string, columns ...string) {
	d.Table.SetTableAlias(alias, columns...)
}

func (d DeleteQuery) WriteSQL(ctx context.Context, w io.StringWriter, dl bob.Dialect, start int) ([]any, error) {
	var err error

	if ctx, err = d.RunContextualMods(ctx, &d); err != nil {
		return nil, err
	}

	writer := queryWriter{
		ctx:   ctx,
		w:     w,
		start: start,
	}

	if len(d.With.CTEs) > 0 {
		args, err := d.With.WriteSQL(ctx, w, dl, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(args)
		_, _ = w.WriteString("\n")
	}

	_, _ = w.WriteString("DELETE FROM ")

	if d.Only {
		_, _ = w.WriteString("ONLY ")
	}

	tableArgs, err := d.Table.WriteSQL(ctx, w, dl, writer.argPos())
	if err != nil {
		return nil, err
	}
	writer.appendArgs(tableArgs)

	if d.TableRef.Expression != nil {
		_, _ = w.WriteString("\nUSING ")
		usingArgs, err := d.TableRef.WriteSQL(ctx, w, dl, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(usingArgs)
	}

	if len(d.Where.Conditions) > 0 {
		_, _ = w.WriteString("\nWHERE ")
		if err := writer.writeSliceAny(d.Where.Conditions, " AND "); err != nil {
			return nil, err
		}
	}

	if len(d.Returning.Expressions) > 0 {
		_, _ = w.WriteString("\nRETURNING ")
		if err := writer.writeSliceAny(d.Returning.Expressions, ", "); err != nil {
			return nil, err
		}
	}

	return writer.args, nil
}
