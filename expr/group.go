package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// Multiple expressions that will be group together as a single expression
type group []bob.Expression

func (g group) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	err := g.WriteSQLTo(ctx, w, d, start, &args)
	return args, err
}

func (g group) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	if len(g) == 0 {
		return bob.ExpressIfTo(ctx, w, d, start, null, true, openPar, closePar, args)
	}

	return bob.ExpressSliceTo(ctx, w, d, start, g, openPar, commaSpace, closePar, args)
}
