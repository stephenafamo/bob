package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// OverridingType represents the OVERRIDING type for INSERT actions (used in both INSERT and MERGE)
type OverridingType string

// OverridingType constants for OVERRIDING clause
const (
	OverridingSystem OverridingType = "SYSTEM"
	OverridingUser   OverridingType = "USER"
)

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-insert.html
type InsertQuery struct {
	clause.With
	Overriding OverridingType
	clause.TableRef
	clause.Values
	clause.Conflict
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*InsertQuery]
}

func (i *InsertQuery) SetTargetTable(table any) {
	i.TableRef.SetTable(table)
}

func (i *InsertQuery) SetTargetTableAlias(alias string, columns ...string) {
	i.TableRef.SetTableAlias(alias, columns...)
}

func (i *InsertQuery) SetOverriding(overriding string) {
	i.Overriding = OverridingType(overriding)
}

func (i *InsertQuery) SetQuery(q bob.Query) {
	i.Values.Query = q
}

func (i InsertQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var err error

	if ctx, err = i.RunContextualMods(ctx, &i); err != nil {
		return nil, err
	}

	writer := queryWriter{
		ctx:   ctx,
		w:     w,
		start: start,
	}

	if len(i.With.CTEs) > 0 {
		args, err := i.With.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(args)
		_, _ = w.WriteString("\n")
	}

	_, _ = w.WriteString("INSERT INTO ")
	if isSimpleTableRef(i.TableRef) {
		if err := writer.writeAny(i.TableRef.Expression); err != nil {
			return nil, err
		}
	} else {
		tableArgs, err := i.TableRef.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(tableArgs)
	}

	if i.Overriding != "" {
		_, _ = w.WriteString("\nOVERRIDING ")
		_, _ = w.WriteString(string(i.Overriding))
		_, _ = w.WriteString(" VALUE")
	}

	_, _ = w.WriteString("\n")
	switch {
	case i.Values.Query != nil:
		valArgs, err := i.Values.Query.WriteQuery(ctx, w, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(valArgs)
	case len(i.Values.Vals) > 0:
		_, _ = w.WriteString("VALUES ")
		for rowIndex, row := range i.Values.Vals {
			if rowIndex > 0 {
				_, _ = w.WriteString(", ")
			}
			_, _ = w.WriteString("(")
			if err := writer.writeSliceExpr(row, ", "); err != nil {
				return nil, err
			}
			_, _ = w.WriteString(")")
		}
	default:
		_, _ = w.WriteString("DEFAULT VALUES")
	}

	if i.Conflict.Expression != nil {
		_, _ = w.WriteString("\n")
		conflictArgs, err := i.Conflict.Expression.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(conflictArgs)
	}

	if len(i.Returning.Expressions) > 0 {
		_, _ = w.WriteString("\nRETURNING ")
		if err := writer.writeSliceAny(i.Returning.Expressions, ", "); err != nil {
			return nil, err
		}
	}

	_, _ = w.WriteString("\n")
	return writer.args, nil
}
