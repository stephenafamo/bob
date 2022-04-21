package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
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

func (o With) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	prefix := "WITH\n"
	if o.Recursive {
		prefix = "WITH RECURSIVE\n"
	}
	return query.ExpressSlice(w, d, start, o.CTEs, prefix, ",\n", "")
}
