package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal"
)

// columnsAssignment represents (columns...) = [ROW] (values...) or (columns...) = (subquery).
// Exactly one of query or values is set, depending on ToQuery vs ToExprs/ToRow.
type columnsAssignment struct {
	cols   []bob.Expression
	query  bob.Query // ToQuery: subquery on the right-hand side
	values []any     // ToExprs / ToRow: expressions on the right-hand side
	isRow  bool
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

	valPrefix := "("
	if a.isRow {
		valPrefix = "ROW ("
	}

	valArgs, err := bob.ExpressSlice(ctx, w, d, start+len(colArgs), a.values, valPrefix, ", ", ")")
	if err != nil {
		return nil, err
	}

	return append(colArgs, valArgs...), nil
}

// SetCols is a reusable helper for PostgreSQL tuple assignments:
// (columns...) = ROW(...) | (values...) | (subquery)
type SetCols[Q interface{ AppendSet(clauses ...any) }] struct {
	columns []string
}

// NewSetCols creates a reusable tuple-assignment builder for SET clauses.
// It can be used by UPDATE queries, INSERT ... ON CONFLICT DO UPDATE,
// and MERGE UPDATE actions.
func NewSetCols[Q interface{ AppendSet(clauses ...any) }](columns ...string) SetCols[Q] {
	return SetCols[Q]{columns: columns}
}

// ToRow sets columns to ROW of expressions: (columns...) = ROW (expressions...)
func (s SetCols[Q]) ToRow(values ...bob.Expression) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		q.AppendSet(columnsAssignment{cols: internal.QuoteIdentifiers(s.columns), values: internal.ToAnySlice(values), isRow: true})
	})
}

// ToExprs sets columns to expressions: (columns...) = (expressions...)
func (s SetCols[Q]) ToExprs(values ...bob.Expression) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		q.AppendSet(columnsAssignment{cols: internal.QuoteIdentifiers(s.columns), values: internal.ToAnySlice(values)})
	})
}

// ToQuery sets columns from a subquery: (columns...) = (subquery)
func (s SetCols[Q]) ToQuery(query bob.Query) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		q.AppendSet(columnsAssignment{cols: internal.QuoteIdentifiers(s.columns), query: query})
	})
}
