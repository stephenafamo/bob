package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// deleteState tries to represent the DELETE query structure as documented in
// https://www.postgresql.org/docs/current/sql-delete.html
type deleteState struct {
	clause.With
	Only       bool
	Table      clause.TableRef
	UsingItems []clause.TableRef
	clause.WhereCurrentOf
	clause.Where
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[DeleteEditor]
}

// AppendTableRef sets the primary USING from_item on the query.
// If the query has no UsingItems, the new TableRef is set as the primary USING from_item.
// If the query has one UsingItem and it is empty, the new TableRef is set as the primary USING from_item.
// If the query has one UsingItem and it is not empty, the new TableRef is appended to the last UsingItem.
func (d *deleteState) AppendTableRef(using clause.TableRef) {
	using = using.Copy()
	if len(d.UsingItems) == 1 && d.UsingItems[0].Expression == nil {
		if len(d.UsingItems[0].Joins) > 0 {
			using.Joins = append(d.UsingItems[0].Joins, using.Joins...)
		}
		d.UsingItems[0] = using

		return
	}

	d.UsingItems = append(d.UsingItems, using)
}

// AppendJoin satisfies Joinable for JoinChain[*deleteState].
// When UsingItems is non-empty, the join is appended to the last USING item (e.g. after um.From).
// Otherwise the new UsingItem is appended with an empty TableRef and the join is appended to it.
func (d *deleteState) AppendJoin(j clause.Join) {
	if len(d.UsingItems) == 0 {
		d.UsingItems = append(d.UsingItems, clause.TableRef{})
	}

	d.UsingItems[len(d.UsingItems)-1].AppendJoin(j)
}

func (d deleteState) WriteSQL(ctx context.Context, w io.StringWriter, dl bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	if len(d.Mods) != 0 {
		ctx, err = d.resolveMods(ctx)
		if err != nil {
			return nil, err
		}
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

		usingArgs, err := expressFromItems(ctx, w, dl, start+len(args), d.UsingItems)
		if err != nil {
			return nil, err
		}
		args = append(args, usingArgs...)
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
