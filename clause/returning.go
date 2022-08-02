package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type Returning struct {
	Expressions []any
}

func (r *Returning) AppendReturning(columns ...any) {
	r.Expressions = append(r.Expressions, columns...)
}

func (r Returning) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, r.Expressions, "RETURNING ", ", ", "")
}
