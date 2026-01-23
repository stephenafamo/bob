package dialect

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/values.html
type ValuesQuery struct {
	// row constructor list
	// each sub-slice is one set of values wrapped in ROW()
	RowVals []Value

	clause.OrderBy
	clause.Limit
}

type Value []bob.Expression

func (v Value) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, v, "ROW(", ", ", ")")
}

func (v *ValuesQuery) AppendValues(vals ...bob.Expression) {
	if len(vals) == 0 {
		return
	}

	v.RowVals = append(v.RowVals, vals)
}

func (v ValuesQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	w.WriteString("VALUES ")

	// write values
	if len(v.RowVals) == 0 {
		return nil, fmt.Errorf("VALUES query must have at least one value expression")
	}
	valuesArgs, err := bob.ExpressSlice(ctx, w, d, start, v.RowVals, "", ", ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, valuesArgs...)

	orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), v.OrderBy,
		len(v.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	limitArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), v.Limit,
		v.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, limitArgs...)

	w.WriteString("\n")
	return args, nil
}
