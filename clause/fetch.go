package clause

import (
	"context"
	"io"
	"strconv"

	"github.com/twitter-payments/bob"
)

type Fetch struct {
	Count    *int64
	WithTies bool
}

func (f *Fetch) SetFetch(fetch Fetch) {
	*f = fetch
}

func (f Fetch) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if f.Count == nil {
		return nil, nil
	}

	w.Write([]byte("FETCH NEXT "))
	w.Write([]byte(strconv.FormatInt(*f.Count, 10)))
	w.Write([]byte(" ROWS "))

	if f.WithTies {
		w.Write([]byte("WITH TIES"))
	} else {
		w.Write([]byte("ONLY"))
	}

	return nil, nil
}
