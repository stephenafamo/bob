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
	Only       bool
	Table      clause.TableRef
	UsingItems []clause.TableRef
	clause.WhereCurrentOf
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*DeleteQuery]
}

// AppendTableRef sets the primary USING from_item on the query.
// If the query has no UsingItems, the new TableRef is set as the primary USING from_item.
// If the query has one UsingItem and it is empty, the new TableRef is set as the primary USING from_item.
// If the query has one UsingItem and it is not empty, the new TableRef is appended to the last UsingItem.
func (d *DeleteQuery) AppendTableRef(using clause.TableRef) {
	if len(d.UsingItems) == 1 && d.UsingItems[0].Expression == nil {
		if len(d.UsingItems[0].Joins) > 0 {
			using.Joins = append(d.UsingItems[0].Joins, using.Joins...)
		}
		d.UsingItems[0] = using

		return
	}

	d.UsingItems = append(d.UsingItems, using)
}

// AppendJoin satisfies Joinable for JoinChain[*DeleteQuery].
// When UsingItems is non-empty, the join is appended to the last USING item (e.g. after um.From).
// Otherwise the new UsingItem is appended with an empty TableRef and the join is appended to it.
func (d *DeleteQuery) AppendJoin(j clause.Join) {
	if len(d.UsingItems) == 0 {
		d.UsingItems = append(d.UsingItems, clause.TableRef{})
	}

	d.UsingItems[len(d.UsingItems)-1].AppendJoin(j)
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

	if len(d.UsingItems) > 0 {
		w.WriteString("\nUSING ")

		args, err = writeFromItemList(ctx, w, dl, start+len(args), args, d.UsingItems)
		if err != nil {
			return nil, err
		}
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
