package builder

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

func (e Builder[T, B]) Group(exps ...any) T {
	return e.X(group(exps))
}

// Multiple expressions that will be group together as a single expression
type group []any

func (g group) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, g, "(", ", ", ")")
}
