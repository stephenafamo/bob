package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	clause "github.com/stephenafamo/bob/clause"
)

// UpdateQuery tries to represent the UPDATE query structure as documented in
// https://www.postgresql.org/docs/current/sql-update.html
type UpdateQuery struct {
	clause.With
	Only  bool
	Table clause.TableRef
	clause.Set
	clause.TableRef
	FromItems []clause.TableRef
	clause.WhereCurrentOf
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*UpdateQuery]
}

func (u *UpdateQuery) SetTable(table any) {
	u.TableRef.SetTable(table)
	u.FromItems = nil
}

func (u *UpdateQuery) AppendTableRef(from clause.TableRef) {
	u.FromItems = append(u.FromItems, from)
}

func (u UpdateQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	if ctx, err = u.RunContextualMods(ctx, &u); err != nil {
		return nil, err
	}

	withArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.WriteString("UPDATE ")

	if u.Only {
		w.WriteString("ONLY ")
	}

	tableArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), u.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	setArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), u.Set, true, " SET\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	hasFrom := u.TableRef.Expression != nil || len(u.FromItems) > 0
	if hasFrom {
		w.WriteString("\nFROM ")

		if u.TableRef.Expression != nil {
			fromArgs, err := bob.Express(ctx, w, d, start+len(args), u.TableRef)
			if err != nil {
				return nil, err
			}
			args = append(args, fromArgs...)
		}

		if len(u.FromItems) > 0 {
			prefix := ""
			if u.TableRef.Expression != nil {
				prefix = ", "
			}

			itemArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), u.FromItems, prefix, ", ", "")
			if err != nil {
				return nil, err
			}
			args = append(args, itemArgs...)
		}
	}

	whereArgs, err := clause.WriteWhereAndCurrentOf(ctx, w, d, start+len(args), u.Where, u.WhereCurrentOf)
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	retArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), u.Returning,
		len(u.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}
