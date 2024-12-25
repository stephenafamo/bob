package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/select.html
type SelectQuery struct {
	hints
	modifiers[any]
	into any

	clause.With
	clause.SelectList
	clause.From
	clause.Where
	clause.GroupBy
	clause.Having
	clause.Windows

	clause.Combines
	clause.OrderBy
	clause.Limit
	clause.Offset
	clause.For
	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*SelectQuery]
}

func (s *SelectQuery) SetInto(i any) {
	s.into = i
}

func (s SelectQuery) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any
	var err error

	if ctx, err = s.RunContextualMods(ctx, &s); err != nil {
		return nil, err
	}

	withArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.With,
		len(s.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("SELECT "))

	// no optimizer hint args
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), s.hints,
		len(s.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(ctx, w, d, start+len(args), s.modifiers,
		len(s.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	selArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.SelectList, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, selArgs...)

	fromArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.From, s.From.Table != nil, "\nFROM ", "")
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

	combineArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args),
		s.Combines.Queries, "\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combineArgs...)

	orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.OrderBy,
		len(s.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = bob.ExpressIf(ctx, w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	_, err = bob.ExpressIf(ctx, w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	forArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.For,
		s.For.Strength != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, forArgs...)

	intoArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), s.into,
		s.into != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, intoArgs...)

	w.Write([]byte("\n"))
	return args, nil
}
