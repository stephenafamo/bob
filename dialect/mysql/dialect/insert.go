package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.4/en/insert.html
type insertState struct {
	hints
	modifiers[string]
	clause.TableRef
	clause.Values

	RowAlias    string
	ColumnAlias []string

	Sets               []Set
	DuplicateKeyUpdate clause.Set

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[InsertEditor]
}

func (i insertState) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	var err error

	if len(i.Mods) != 0 {
		ctx, err = i.resolveMods(ctx)
		if err != nil {
			return nil, err
		}
	}

	w.WriteString("INSERT ")

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
	if len(i.Sets) != 0 {
		// SET supplies its own column names, including on model inserts.
		i.TableRef.Columns = nil
	}
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), i.TableRef, true, "INTO ", " ")
	if err != nil {
		return nil, err
	}

	// Either this or the values will get expressed
	setArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), i.Sets, "\nSET ", ",\n", "\n")
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
		w.WriteString("\nAS ")
		d.WriteQuoted(w, i.RowAlias)

		if len(i.ColumnAlias) > 0 {
			w.WriteString("(")
			for k, cAlias := range i.ColumnAlias {
				if k != 0 {
					w.WriteString(", ")
				}

				d.WriteQuoted(w, cAlias)
			}
			w.WriteString(")")
		}
	}

	updateArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), expr.PrepareSetAssignments(i.DuplicateKeyUpdate.Set),
		"\nON DUPLICATE KEY UPDATE\n", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, updateArgs...)

	w.WriteString("\n")
	return args, nil
}
