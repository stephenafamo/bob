package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// Multiple expressions that will be group together as a single expression
type group []bob.Expression

func (g group) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(g) == 0 {
		return bob.ExpressIf(ctx, w, d, start, null, true, openPar, closePar)
	}

	return bob.ExpressSlice(ctx, w, d, start, g, openPar, commaSpace, closePar)
}
