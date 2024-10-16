package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type or struct {
	action string
}

func (o *or) SetOr(to string) {
	o.action = to
}

func (o or) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressIf(ctx, w, d, start, o.action, o.action != "", " OR ", "")
}
