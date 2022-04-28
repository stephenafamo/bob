package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

func Group(exps ...any) query.Expression {
	return group(exps)
}

// Multiple expressions that will be group together as a single expression
type group []any

func (g group) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, g, "(", ", ", ")")
}
