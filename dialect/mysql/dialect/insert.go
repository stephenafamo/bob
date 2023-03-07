package dialect

import (
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
	DuplicateKeyUpdate []Set
}

func (i InsertQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any
	var err error

	w.Write([]byte("INSERT "))

	// no optimizer hint args
	_, err = bob.ExpressIf(w, d, start+len(args), i.hints,
		len(i.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(w, d, start+len(args), i.modifiers,
		len(i.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	// no expected table args
	_, err = bob.ExpressIf(w, d, start+len(args), i.Table, true, "INTO ", " ")
	if err != nil {
		return nil, err
	}

	// no partition args
	_, err = bob.ExpressIf(w, d, start+len(args), i.partitions,
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
	valArgs, err := bob.ExpressSlice(w, d, start+len(args), i.Sets, "\nSET ", "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	// Either this or SET will get expressed
	setArgs, err := bob.ExpressIf(w, d, start+len(args), i.Values, len(i.Sets) == 0, "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

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

	// Either this or the values will get expressed
	updateArgs, err := bob.ExpressSlice(w, d, start+len(args), i.DuplicateKeyUpdate,
		"\nON DUPLICATE KEY UPDATE\n", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, updateArgs...)

	w.Write([]byte("\n"))
	return args, nil
}
