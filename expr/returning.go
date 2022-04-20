package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type Returning struct {
	Expressions []any
}

func (r *Returning) AppendReturning(columns ...any) {
	r.Expressions = append(r.Expressions, columns...)
}

func (r Returning) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, r.Expressions, "RETURNING ", ", ", "")
}
