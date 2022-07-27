package clause

import (
	"io"

	"github.com/stephenafamo/bob/query"
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

func (w With) WriteSQL(wr io.Writer, d query.Dialect, start int) ([]any, error) {
	prefix := "WITH\n"
	if w.Recursive {
		prefix = "WITH RECURSIVE\n"
	}
	return query.ExpressSlice(wr, d, start, w.CTEs, prefix, ",\n", "")
}
