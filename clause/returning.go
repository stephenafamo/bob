package clause

import (
	"context"
	"io"

	"github.com/twitter-payments/bob"
)

type Returning struct {
	Expressions []any
}

func (r *Returning) HasReturning() bool {
	return len(r.Expressions) > 0
}

func (r *Returning) AppendReturning(columns ...any) {
	r.Expressions = append(r.Expressions, columns...)
}

func (r Returning) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, r.Expressions, "RETURNING ", ", ", "")
}
