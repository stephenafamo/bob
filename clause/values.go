package clause

import (
	"context"
	"io"

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

func (v Value) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, v, "(", ", ", ")")
}

func (v *Values) AppendValues(vals ...bob.Expression) {
	if len(vals) == 0 {
		return
	}

	v.Vals = append(v.Vals, vals)
}

func (v Values) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	// If a query is present, use it
	if v.Query != nil {
		return v.Query.WriteQuery(ctx, w, start)
	}

	// If values are present, use them
	if len(v.Vals) > 0 {
		return bob.ExpressSlice(ctx, w, d, start, v.Vals, "VALUES ", ", ", "")
	}

	// If no value was present, use default value
	w.Write([]byte("DEFAULT VALUES"))
	return nil, nil
}
