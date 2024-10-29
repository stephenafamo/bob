package dialect

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/insert.html
type InsertQuery struct {
	hints
	modifiers[string]
	partitions
	clause.Values

	Table              any
	Columns            []string
	RowAlias           string
	ColumnAlias        []string
	Sets               []Set
	DuplicateKeyUpdate clause.Set

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*InsertQuery]
}

func (i InsertQuery) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any
	var err error

	if ctx, err = i.RunContextualMods(ctx, &i); err != nil {
		return nil, err
	}

	w.Write([]byte("INSERT "))

	// no optimizer hint args
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), i.hints,
		len(i.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), i.modifiers,
		len(i.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	// no expected table args
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), i.Table, true, "INTO ", " ")
	if err != nil {
		return nil, err
	}

	// no partition args
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), i.partitions,
		len(i.partitions.partitions) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	// No columns args
	if len(i.Columns) > 0 {
		w.Write([]byte(" ("))
		for k, cAlias := range i.Columns {
			if k != 0 {
				w.Write([]byte(", "))
			}

			d.WriteQuoted(w, cAlias)
		}
		w.Write([]byte(")"))
	}

	// Either this or the values will get expressed
	setArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), i.Sets, "\nSET ", "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	// Either this or SET will get expressed
	valArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), i.Values, len(i.Sets) == 0, "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	// The aliases
	if i.RowAlias != "" {
		_, err = fmt.Fprintf(w, "\nAS %s", i.RowAlias)
		if err != nil {
			return nil, err
		}

		if len(i.ColumnAlias) > 0 {
			w.Write([]byte("("))
			for k, cAlias := range i.ColumnAlias {
				if k != 0 {
					w.Write([]byte(", "))
				}

				d.WriteQuoted(w, cAlias)
			}
			w.Write([]byte(")"))
		}
	}

	updateArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), i.DuplicateKeyUpdate.Set,
		"\nON DUPLICATE KEY UPDATE\n", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, updateArgs...)

	w.Write([]byte("\n"))
	return args, nil
}
