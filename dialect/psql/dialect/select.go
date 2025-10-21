package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-select.html
type SelectQuery struct {
	clause.With
	clause.SelectList
	Distinct
	clause.TableRef
	clause.Where
	clause.GroupBy
	clause.Having
	clause.Windows
	clause.Combines
	clause.OrderBy
	clause.Limit
	clause.Offset
	clause.Fetch
	clause.Locks

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*SelectQuery]

	CombinedOrder  clause.OrderBy
	CombinedLimit  clause.Limit
	CombinedFetch  clause.Fetch
	CombinedOffset clause.Offset
}

func (s SelectQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	if ctx, err = s.RunContextualMods(ctx, &s); err != nil {
		return nil, err
	}

	withArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.With,
		len(s.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	needsParens := false
	if len(s.Combines.Queries) > 0 &&
		(len(s.OrderBy.Expressions) > 0 ||
			s.Limit.Count != nil ||
			s.Offset.Count != nil ||
			s.Fetch.Count != nil ||
			len(s.Locks.Locks) > 0) {
		w.WriteString("(")
		needsParens = true
	}

	w.WriteString("SELECT ")

	distinctArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Distinct,
		s.Distinct.On != nil, "", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, distinctArgs...)

	selArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.SelectList, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, selArgs...)

	fromArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.TableRef, s.TableRef.Expression != nil, "\nFROM ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Where,
		len(s.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	groupByArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.GroupBy,
		len(s.GroupBy.Groups) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, groupByArgs...)

	havingArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Having,
		len(s.Having.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, havingArgs...)

	windowArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Windows,
		len(s.Windows.Windows) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, windowArgs...)

	orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.OrderBy,
		len(s.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	limitArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, limitArgs...)

	offsetArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, offsetArgs...)

	fetchArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.Fetch,
		s.Fetch.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fetchArgs...)

	lockArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), s.Locks.Locks,
		"\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, lockArgs...)

	if needsParens {
		w.WriteString(")")
	}

	combineArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args),
		s.Combines.Queries, "\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combineArgs...)

	combinedOrderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.CombinedOrder,
		len(s.CombinedOrder.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combinedOrderArgs...)

	combinedLimitArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.CombinedLimit,
		s.CombinedLimit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combinedLimitArgs...)

	combinedOffsetArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.CombinedOffset,
		s.CombinedOffset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combinedOffsetArgs...)

	combinedFetchArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.CombinedFetch,
		s.CombinedFetch.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combinedFetchArgs...)

	w.WriteString("\n")
	return args, nil
}
