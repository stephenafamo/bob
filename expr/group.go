package expr

import (
	"io"

	"github.com/stephenafamo/bob"
)

// Multiple expressions that will be group together as a single expression
type group []any

func (g group) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, g, "(", ", ", ")")
}
