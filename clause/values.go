package clause

import (
	"context"
	"io"
	"slices"

	"github.com/stephenafamo/bob"
)

type Values struct {
	// Query takes the highest priority
	// If present, will attempt to insert from this query
	Query bob.Query

	// for multiple inserts
	// each sub-slice is one set of values
	Vals []Value
}

type Value []bob.Expression

func (v Value) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return v.appendSQL(ctx, w, d, start, nil)
}

// appendSQL adds this row's arguments to the statement's argument list.
func (v Value) appendSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args []any) ([]any, error) {
	if len(v) == 0 {
		return args, nil
	}
	w.WriteString("(")
	for i, cell := range v {
		if i > 0 {
			w.WriteString(", ")
		}
		cellArgs, err := bob.Express(ctx, w, d, start+len(args), cell)
		if err != nil {
			return args, err
		}
		args = append(args, cellArgs...)
	}
	w.WriteString(")")
	return args, nil
}

func (v *Values) AppendValues(vals ...bob.Expression) {
	if len(vals) == 0 {
		return
	}

	v.Vals = append(v.Vals, slices.Clone(vals))
}

// AppendRows adds the nonempty rows, with one backing array for their cells.
func (v *Values) AppendRows(rows ...[]bob.Expression) {
	var count, size int
	for _, row := range rows {
		if len(row) > 0 {
			count++
			size += len(row)
		}
	}
	if count == 0 {
		return
	}
	v.Vals = slices.Grow(v.Vals, count)
	cells := make([]bob.Expression, size)
	for _, row := range rows {
		if n := copy(cells, row); n != 0 {
			v.Vals = append(v.Vals, cells[:n:n])
			cells = cells[n:]
		}
	}
}

func (v Values) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	// If a query is present, use it
	if v.Query != nil {
		return bob.WriteQuery(ctx, w, start, v.Query)
	}

	if len(v.Vals) == 0 {
		w.WriteString("DEFAULT VALUES")
		return nil, nil
	}

	var args []any
	w.WriteString("VALUES ")
	for i, row := range v.Vals {
		if i > 0 {
			w.WriteString(", ")
		}
		next, err := row.appendSQL(ctx, w, d, start, args)
		if err != nil {
			return args, err // Only return arguments from completed rows.
		}
		args = next
	}
	return args, nil
}
