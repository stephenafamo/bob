package dialect

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// Trying to represent the values query structure as doumented in
// https://www.postgresql.org/docs/current/sql-values.html
type ValuesQuery struct {
	// rows of VALUES query
	// each sub-slice is one set of values
	RowVals []RowValue

	clause.OrderBy
	clause.Limit
	clause.Offset
	clause.Fetch
}

type RowValue []bob.Expression

func (v RowValue) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, v, "(", ", ", ")")
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

	offsetArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), v.Offset,
		v.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, offsetArgs...)

	fetchArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), v.Fetch,
		v.Fetch.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fetchArgs...)

	w.WriteString("\n")
	return args, nil
}
