package builder

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

// Comma separated list of arguments
func (e Builder[T, B]) Arg(vals ...any) T {
	return e.X(args{vals: vals})
}

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
