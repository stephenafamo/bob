package builder

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

// Multiple expressions that will be group together as a single expression
type group []any

func (g group) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, g, "(", ", ", ")")
}
