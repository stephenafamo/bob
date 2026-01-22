package dialect

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
)

// Trying to represent the row value as documented in
// https://www.sqlite.org/rowvalue.html
type ValuesQuery struct {
	RowVals []RowValue
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
	args, err = bob.ExpressSlice(ctx, w, d, start, v.RowVals, "", ", ", "")
	if err != nil {
		return nil, err
	}

	w.WriteString("\n")
	return args, nil
}
