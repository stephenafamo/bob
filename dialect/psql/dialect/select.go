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

	if ctx, err = s.RunContextualMods(ctx, &s); err != nil {
		return nil, err
	}

	writer := queryWriter{
		ctx:   ctx,
		w:     w,
		start: start,
	}

	if len(s.With.CTEs) > 0 {
		args, err := s.With.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(args)
		_, _ = w.WriteString("\n")
	}

	needsParens := false
	if len(s.Combines.Queries) > 0 &&
		(len(s.OrderBy.Expressions) > 0 ||
			s.Limit.Count != nil ||
			s.Offset.Count != nil ||
			s.Fetch.Count != nil ||
			len(s.Locks.Locks) > 0) {
		_, _ = w.WriteString("(")
		needsParens = true
	}

	_, _ = w.WriteString("SELECT ")

	if s.Distinct.On != nil {
		_, _ = w.WriteString("DISTINCT")
		if len(s.Distinct.On) > 0 {
			_, _ = w.WriteString(" ON (")
			if err := writer.writeSliceAny(s.Distinct.On, ", "); err != nil {
				return nil, err
			}
			_, _ = w.WriteString(")")
		}
		_, _ = w.WriteString(" ")
	}

	_, _ = w.WriteString("\n")
	allCols := append([]any(nil), s.SelectList.Columns...)
	allCols = append(allCols, s.SelectList.PreloadColumns...)
	if len(allCols) == 0 {
		_, _ = w.WriteString("*")
	} else if err := writer.writeSliceAny(allCols, ", "); err != nil {
		return nil, err
	}

	if s.TableRef.Expression != nil {
		_, _ = w.WriteString("\nFROM ")
		args, err := s.TableRef.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(args)
	}

	if len(s.Where.Conditions) > 0 {
		_, _ = w.WriteString("\nWHERE ")
		if err := writer.writeSliceAny(s.Where.Conditions, " AND "); err != nil {
			return nil, err
		}
	}

	if len(s.GroupBy.Groups) > 0 {
		_, _ = w.WriteString("\nGROUP BY ")
		if s.GroupBy.Distinct {
			_, _ = w.WriteString("DISTINCT ")
		}
		if err := writer.writeSliceAny(s.GroupBy.Groups, ", "); err != nil {
			return nil, err
		}
		if s.GroupBy.With != "" {
			_, _ = w.WriteString(" WITH ")
			_, _ = w.WriteString(s.GroupBy.With)
		}
	}

	if len(s.Having.Conditions) > 0 {
		_, _ = w.WriteString("\nHAVING ")
		if err := writer.writeSliceAny(s.Having.Conditions, " AND "); err != nil {
			return nil, err
		}
	}

	if len(s.Windows.Windows) > 0 {
		_, _ = w.WriteString("\nWINDOW ")
		if err := writer.writeSliceExpr(s.Windows.Windows, ", "); err != nil {
			return nil, err
		}
	}

	if len(s.OrderBy.Expressions) > 0 {
		_, _ = w.WriteString("\nORDER BY ")
		if err := writer.writeOrderExprs(s.OrderBy.Expressions); err != nil {
			return nil, err
		}
	}

	if s.Limit.Count != nil {
		_, _ = w.WriteString("\nLIMIT ")
		if err := writer.writeAny(s.Limit.Count); err != nil {
			return nil, err
		}
	}

	if s.Offset.Count != nil {
		_, _ = w.WriteString("\nOFFSET ")
		if err := writer.writeAny(s.Offset.Count); err != nil {
			return nil, err
		}
	}

	if s.Fetch.Count != nil {
		_, _ = w.WriteString("\nFETCH NEXT ")
		if err := writer.writeAny(s.Fetch.Count); err != nil {
			return nil, err
		}
		if s.Fetch.WithTies {
			_, _ = w.WriteString(" ROWS WITH TIES")
		} else {
			_, _ = w.WriteString(" ROWS ONLY")
		}
	}

	for _, lock := range s.Locks.Locks {
		_, _ = w.WriteString("\n")
		if err := writer.writeExpression(lock); err != nil {
			return nil, err
		}
	}

	if needsParens {
		_, _ = w.WriteString(")")
	}

	for _, combine := range s.Combines.Queries {
		_, _ = w.WriteString("\n")
		args, err := combine.WriteSQL(ctx, w, d, writer.argPos())
		if err != nil {
			return nil, err
		}
		writer.appendArgs(args)
	}

	if len(s.CombinedOrder.Expressions) > 0 {
		_, _ = w.WriteString("\nORDER BY ")
		if err := writer.writeOrderExprs(s.CombinedOrder.Expressions); err != nil {
			return nil, err
		}
	}

	if s.CombinedLimit.Count != nil {
		_, _ = w.WriteString("\nLIMIT ")
		if err := writer.writeAny(s.CombinedLimit.Count); err != nil {
			return nil, err
		}
	}

	if s.CombinedOffset.Count != nil {
		_, _ = w.WriteString("\nOFFSET ")
		if err := writer.writeAny(s.CombinedOffset.Count); err != nil {
			return nil, err
		}
	}

	if s.CombinedFetch.Count != nil {
		_, _ = w.WriteString("\nFETCH NEXT ")
		if err := writer.writeAny(s.CombinedFetch.Count); err != nil {
			return nil, err
		}
		if s.CombinedFetch.WithTies {
			_, _ = w.WriteString(" ROWS WITH TIES")
		} else {
			_, _ = w.WriteString(" ROWS ONLY")
		}
	}

	_, _ = w.WriteString("\n")
	return writer.args, nil
}
