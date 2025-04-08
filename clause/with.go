package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type With struct {
	Recursive bool
	CTEs      []bob.Expression
}

func (w *With) AppendCTE(cte bob.Expression) {
	w.CTEs = append(w.CTEs, cte)
}

func (w *With) SetRecursive(r bool) {
	w.Recursive = r
}

func (w With) WriteSQL(ctx context.Context, wr io.Writer, d bob.Dialect, start int) ([]any, error) {
	prefix := "WITH\n"
	if w.Recursive {
		prefix = "WITH RECURSIVE\n"
	}
	return bob.ExpressSlice(ctx, wr, d, start, w.CTEs, prefix, ",\n", "")
}
