package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Fetch struct {
	Count    any
	WithTies bool
}

func (f *Fetch) SetFetch(fetch Fetch) {
	*f = fetch
}

func (f Fetch) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if f.Count == nil {
		return nil, nil
	}

	suffix := " ROWS ONLY"
	if f.WithTies {
		suffix = " ROWS WITH TIES"
	}

	return bob.ExpressIf(ctx, w, d, start, f.Count, true, "FETCH NEXT ", suffix)
}
