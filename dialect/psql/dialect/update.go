package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	clause "github.com/stephenafamo/bob/clause"
)

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-update.html
type UpdateQuery struct {
	clause.With
	Only  bool
	Table clause.TableRef
	clause.Set
	clause.TableRef
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*UpdateQuery]
}

func (u *UpdateQuery) SetTargetOnly(only bool) {
	u.Table.SetOnly(only)
}

func (u *UpdateQuery) SetTargetTable(table any) {
	u.Table.SetTable(table)
}

func (u *UpdateQuery) SetTargetTableAlias(alias string, columns ...string) {
	u.Table.SetTableAlias(alias, columns...)
}

func (u UpdateQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var err error

	if ctx, err = u.RunContextualMods(ctx, &u); err != nil {
		return nil, err
	}

	writer := queryWriter{
		ctx:   ctx,
		w:     w,
		start: start,
	}

	if len(u.With.CTEs) > 0 {
		args, err := u.With.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(args)
		_, _ = w.WriteString("\n")
	}

	_, _ = w.WriteString("UPDATE ")

	if u.Only {
		_, _ = w.WriteString("ONLY ")
	}

	tableArgs, err := u.Table.WriteSQL(ctx, w, d, writer.argPos())
	if err != nil {
		return nil, err
	}
	writer.appendArgs(tableArgs)

	_, _ = w.WriteString(" SET\n")
	setArgs, err := u.Set.WriteSQL(ctx, w, d, writer.argPos())
	if err != nil {
		return nil, err
	}
	writer.appendArgs(setArgs)

	if u.TableRef.Expression != nil {
		_, _ = w.WriteString("\nFROM ")
		fromArgs, err := u.TableRef.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(fromArgs)
	}

	if len(u.Where.Conditions) > 0 {
		_, _ = w.WriteString("\n")
		whereArgs, err := u.Where.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(whereArgs)
	}

	if len(u.Returning.Expressions) > 0 {
		_, _ = w.WriteString("\n")
		retArgs, err := u.Returning.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(retArgs)
	}

	return writer.args, nil
}
