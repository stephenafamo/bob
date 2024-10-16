package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Having struct {
	Conditions []any
}

func (h *Having) AppendHaving(e ...any) {
	h.Conditions = append(h.Conditions, e...)
}

func (h Having) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.ExpressSlice(ctx, w, d, start, h.Conditions, "HAVING ", " AND ", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
