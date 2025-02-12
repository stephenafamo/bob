package clause

import (
	"context"
	"io"

	"github.com/twitter-payments/bob"
)

type With struct {
	Recursive bool
	CTEs      []CTE
}

func (w *With) AppendWith(cte CTE) {
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
