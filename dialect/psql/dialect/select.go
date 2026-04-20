package dialect

import (
	"context"
	"fmt"
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

	shared selectShared
}

func (s SelectQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	err := s.WriteSQLTo(ctx, w, d, start, &args)
	return args, err
}

func (s SelectQuery) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	var err error
	baseLen := len(*args)
	nextStart := func() int {
		return start + len(*args) - baseLen
	}

	if ctx, err = s.RunContextualMods(ctx, &s); err != nil {
		return err
	}

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.With,
		len(s.With.CTEs) > 0, "\n", "", args); err != nil {
		return err
	}

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

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.Distinct,
		s.Distinct.On != nil, "", " ", args); err != nil {
		return err
	}

	w.WriteString("\n")
	if err := writeSelectListTo(ctx, w, d, nextStart(), s.SelectList, args); err != nil {
		return err
	}

	if s.TableRef.Expression != nil {
		w.WriteString("\nFROM ")
		if err := writeTableRefTo(ctx, w, d, nextStart(), s.TableRef, args); err != nil {
			return err
		}
	}

	if len(s.Where.Conditions) > 0 {
		w.WriteString("\n")
		if err := writeWhereTo(ctx, w, d, nextStart(), s.Where, args); err != nil {
			return err
		}
	}

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.GroupBy,
		len(s.GroupBy.Groups) > 0, "\n", "", args); err != nil {
		return err
	}

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.Having,
		len(s.Having.Conditions) > 0, "\n", "", args); err != nil {
		return err
	}

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.Windows,
		len(s.Windows.Windows) > 0, "\n", "", args); err != nil {
		return err
	}

	if len(s.OrderBy.Expressions) > 0 {
		w.WriteString("\n")
		if err := writeOrderByTo(ctx, w, d, nextStart(), s.OrderBy, args); err != nil {
			return err
		}
	}

	if s.Limit.Count != nil {
		w.WriteString("\n")
		if err := writeLimitTo(ctx, w, d, nextStart(), s.Limit, args); err != nil {
			return err
		}
	}

	if s.Offset.Count != nil {
		w.WriteString("\n")
		if err := writeOffsetTo(ctx, w, d, nextStart(), s.Offset, args); err != nil {
			return err
		}
	}

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.Fetch,
		s.Fetch.Count != nil, "\n", "", args); err != nil {
		return err
	}

	if err := bob.ExpressSliceTo(ctx, w, d, nextStart(), s.Locks.Locks,
		"\n", "\n", "", args); err != nil {
		return err
	}

	if needsParens {
		w.WriteString(")")
	}

	if err := bob.ExpressSliceTo(ctx, w, d, nextStart(),
		s.Combines.Queries, "\n", "\n", "", args); err != nil {
		return err
	}

	if len(s.CombinedOrder.Expressions) > 0 {
		w.WriteString("\n")
		if err := writeOrderByTo(ctx, w, d, nextStart(), s.CombinedOrder, args); err != nil {
			return err
		}
	}

	if s.CombinedLimit.Count != nil {
		w.WriteString("\n")
		if err := writeLimitTo(ctx, w, d, nextStart(), s.CombinedLimit, args); err != nil {
			return err
		}
	}

	if s.CombinedOffset.Count != nil {
		w.WriteString("\n")
		if err := writeOffsetTo(ctx, w, d, nextStart(), s.CombinedOffset, args); err != nil {
			return err
		}
	}

	if err := bob.ExpressIfTo(ctx, w, d, nextStart(), s.CombinedFetch,
		s.CombinedFetch.Count != nil, "\n", "", args); err != nil {
		return err
	}

	w.WriteString("\n")
	return nil
}

func writeSelectListTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, s clause.SelectList, args *[]any) error {
	baseLen := len(*args)
	nextStart := func() int {
		return start + len(*args) - baseLen
	}

	wrote := false
	if len(s.Columns) > 0 {
		if err := bob.ExpressSliceTo(ctx, w, d, nextStart(), s.Columns, "", ", ", "", args); err != nil {
			return err
		}
		wrote = true
	}

	if len(s.PreloadColumns) > 0 {
		if wrote {
			w.WriteString(", ")
		}
		if err := bob.ExpressSliceTo(ctx, w, d, nextStart(), s.PreloadColumns, "", ", ", "", args); err != nil {
			return err
		}
		wrote = true
	}

	if !wrote {
		w.WriteString("*")
	}

	return nil
}

func writeWhereTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, wh clause.Where, args *[]any) error {
	return bob.ExpressSliceTo(ctx, w, d, start, wh.Conditions, "WHERE ", " AND ", "", args)
}

func writeOrderByTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, order clause.OrderBy, args *[]any) error {
	baseLen := len(*args)
	nextStart := func() int {
		return start + len(*args) - baseLen
	}

	w.WriteString("ORDER BY ")
	for i, expression := range order.Expressions {
		if i != 0 {
			w.WriteString(", ")
		}

		if err := bob.ExpressTo(ctx, w, d, nextStart(), expression, args); err != nil {
			return err
		}
	}

	return nil
}

func writeLimitTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, limit clause.Limit, args *[]any) error {
	w.WriteString("LIMIT ")
	return bob.ExpressTo(ctx, w, d, start, limit.Count, args)
}

func writeOffsetTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, offset clause.Offset, args *[]any) error {
	w.WriteString("OFFSET ")
	return bob.ExpressTo(ctx, w, d, start, offset.Count, args)
}

func writeTableRefTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, table clause.TableRef, args *[]any) error {
	baseLen := len(*args)
	nextStart := func() int {
		return start + len(*args) - baseLen
	}

	if table.Only {
		w.WriteString("ONLY ")
	}

	if table.Lateral {
		w.WriteString("LATERAL ")
	}

	if err := bob.ExpressTo(ctx, w, d, nextStart(), table.Expression, args); err != nil {
		return err
	}

	if table.WithOrdinality {
		w.WriteString(" WITH ORDINALITY")
	}

	if len(table.Partitions) > 0 {
		if err := bob.ExpressSliceTo(ctx, w, d, nextStart(), table.Partitions, " PARTITION (", ", ", ")", args); err != nil {
			return err
		}
	}

	if table.Alias != "" {
		w.WriteString(" AS ")
		d.WriteQuoted(w, table.Alias)
	}

	if len(table.Columns) > 0 {
		w.WriteString("(")
		for i, alias := range table.Columns {
			if i != 0 {
				w.WriteString(", ")
			}
			d.WriteQuoted(w, alias)
		}
		w.WriteString(")")
	}

	if len(table.IndexHints) > 0 {
		for i, hint := range table.IndexHints {
			if i == 0 {
				w.WriteString("\n")
			} else {
				w.WriteString(" ")
			}
			writeIndexHint(w, d, hint)
		}
	}

	switch {
	case table.IndexedBy == nil:
	case *table.IndexedBy == "":
		w.WriteString(" NOT INDEXED")
	default:
		w.WriteString(fmt.Sprintf(" INDEXED BY %q", *table.IndexedBy))
	}

	for _, join := range table.Joins {
		w.WriteString("\n")
		if err := writeJoinTo(ctx, w, d, nextStart(), join, args); err != nil {
			return err
		}
	}

	return nil
}

func writeJoinTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, join clause.Join, args *[]any) error {
	baseLen := len(*args)
	nextStart := func() int {
		return start + len(*args) - baseLen
	}

	if join.Natural {
		w.WriteString("NATURAL ")
	}

	w.WriteString(join.Type)
	w.WriteString(" ")

	if err := writeTableRefTo(ctx, w, d, nextStart(), join.To, args); err != nil {
		return err
	}

	if len(join.On) > 0 {
		if err := bob.ExpressSliceTo(ctx, w, d, nextStart(), join.On, " ON ", " AND ", "", args); err != nil {
			return err
		}
	}

	for i, col := range join.Using {
		if i == 0 {
			w.WriteString(" USING(")
		} else {
			w.WriteString(", ")
		}

		d.WriteQuoted(w, col)

		if i == len(join.Using)-1 {
			w.WriteString(") ")
		}
	}

	return nil
}

func writeIndexHint(w io.StringWriter, d bob.Dialect, hint clause.IndexHint) {
	if hint.Type == "" {
		return
	}

	w.WriteString(hint.Type)
	w.WriteString(" INDEX ")
	if hint.For != "" {
		w.WriteString(" FOR ")
		w.WriteString(hint.For)
	}

	w.WriteString(" (")
	for i, index := range hint.Indexes {
		if i != 0 {
			w.WriteString(", ")
		}
		w.WriteString(index)
	}
	w.WriteString(")")
}
