package expr

import (
	"io"

	"github.com/stephenafamo/bob"
)

// To pervent unnecessary allocations
const (
	openPar    = "("
	closePar   = ")"
	commaSpace = ", "
)

// Multiple expressions that will be group together as a single expression
type group []any

func (g group) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, g, openPar, commaSpace, closePar)
}
