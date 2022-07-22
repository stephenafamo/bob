package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type args struct {
	vals []any
}

func (a args) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	for k := range a.vals {
		if k > 0 {
			w.Write([]byte(", "))
		}

		d.WriteArg(w, start+k)
	}

	return a.vals, nil
}
