package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal"
)

// columnsAssignment represents (columns...) = [ROW] (values...) or (columns...) = (subquery).
// Exactly one of query or values is set, depending on ToQuery vs ToExprs/ToRow.
type columnsAssignment struct {
	cols      []bob.Expression
	query     bob.Query // ToQuery: subquery on the right-hand side
	values    []any     // ToExprs / ToRow: expressions on the right-hand side
	rowPrefix string    // ToRow only: token before "(", e.g. "ROW" on PostgreSQL
}

func (a columnsAssignment) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	colArgs, err := bob.ExpressSlice(ctx, w, d, start, a.cols, "(", ", ", ") = ")
	if err != nil {
		return nil, err
	}

	if a.query != nil {
		w.WriteString("(")
		valArgs, err := a.query.WriteQuery(ctx, w, start+len(colArgs))
		if err != nil {
			return nil, err
		}
		w.WriteString(")")
		return append(colArgs, valArgs...), nil
	}

	if a.rowPrefix != "" {
		w.WriteString(a.rowPrefix)
		w.WriteString(" ")
	}

	valArgs, err := bob.ExpressSlice(ctx, w, d, start+len(colArgs), a.values, "(", ", ", ")")
	if err != nil {
		return nil, err
	}

	return append(colArgs, valArgs...), nil
}

// SetColsOptions configures tuple-assignment rendering for a [SetCols] builder.
type SetColsOptions struct {
	// RowPrefix is emitted before the value list in ToRow, e.g. "ROW" on PostgreSQL.
	RowPrefix string
}

// SetCols is a reusable helper for tuple assignments in SET clauses:
// (columns...) = ROW(...) | (values...) | (subquery)
type SetCols[Q interface{ AppendSet(clauses ...any) }] struct {
	columns []string
	opts    SetColsOptions
}

// NewSetCols creates a tuple-assignment builder for SET clauses.
// It can be used by UPDATE queries, INSERT ... ON CONFLICT DO UPDATE, and MERGE UPDATE actions.
// Dialect-specific rendering is configured via [SetCols.Options].
func NewSetCols[Q interface{ AppendSet(clauses ...any) }](columns ...string) SetCols[Q] {
	return SetCols[Q]{columns: columns}
}

// Options returns a copy of the builder with the given options applied.
func (s SetCols[Q]) Options(opts SetColsOptions) SetCols[Q] {
	s.opts = opts
	return s
}

// ToRow sets columns to a row of expressions: (columns...) = [prefix] (expressions...)
func (s SetCols[Q]) ToRow(values ...bob.Expression) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		q.AppendSet(columnsAssignment{
			cols:      internal.QuoteIdentifiers(s.columns),
			values:    internal.ToAnySlice(values),
			rowPrefix: s.opts.RowPrefix,
		})
	})
}

// ToExprs sets columns to expressions: (columns...) = (expressions...)
func (s SetCols[Q]) ToExprs(values ...bob.Expression) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		q.AppendSet(columnsAssignment{
			cols:   internal.QuoteIdentifiers(s.columns),
			values: internal.ToAnySlice(values),
		})
	})
}

// ToQuery sets columns from a subquery: (columns...) = (subquery)
func (s SetCols[Q]) ToQuery(query bob.Query) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		q.AppendSet(columnsAssignment{cols: internal.QuoteIdentifiers(s.columns), query: query})
	})
}
