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
	FromItems []clause.TableRef
	clause.WhereCurrentOf
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*UpdateQuery]
}

// AppendTableRef sets the primary FROM from_item on the query.
// If the query has no FromItems, the new TableRef is set as the primary FROM from_item.
// If the query has one FromItem and it is empty, the new TableRef is set as the primary FROM from_item.
// If the query has one FromItem and it is not empty, the new TableRef is appended to the last FromItem.
func (u *UpdateQuery) AppendTableRef(from clause.TableRef) {
	if len(u.FromItems) == 1 && u.FromItems[0].Expression == nil {
		if len(u.FromItems[0].Joins) > 0 {
			from.Joins = append(u.FromItems[0].Joins, from.Joins...)
		}
		u.FromItems[0] = from

		return
	}

	u.FromItems = append(u.FromItems, from)
}

// AppendJoin satisfies Joinable for JoinChain[*UpdateQuery].
// When FromItems is non-empty, the join is appended to the last FROM item (e.g. after um.From).
// Otherwise the new FromItem is appended with an empty TableRef and the join is appended to it.
func (u *UpdateQuery) AppendJoin(j clause.Join) {
	if len(u.FromItems) == 0 {
		u.FromItems = append(u.FromItems, clause.TableRef{})
	}

	u.FromItems[len(u.FromItems)-1].AppendJoin(j)
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

	if len(u.FromItems) > 0 {
		w.WriteString("\nFROM ")

		args, err = writeFromItemList(ctx, w, d, start+len(args), args, u.FromItems)
		if err != nil {
			return nil, err
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
