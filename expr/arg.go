package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal"
)

func Arg(vals ...any) bob.Expression {
	return args{vals: vals}
}

// Like Arg, but wraps in parentheses
func ArgGroup(vals ...any) bob.Expression {
	return args{vals: vals, grouped: true}
}

type args struct {
	vals    []any
	grouped bool
}

func (a args) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if a.grouped {
		w.Write([]byte(openPar))
	}

	if len(a.vals) == 0 {
		w.Write([]byte("NULL"))
	}

	for k := range a.vals {
		if k > 0 {
			w.Write([]byte(commaSpace))
		}

		d.WriteArg(w, start+k)
	}

	if a.grouped {
		w.Write([]byte(closePar))
	}

	return a.vals, nil
}

func ToArgs[T any](vals ...T) bob.Expression {
	return Arg(internal.ToAnySlice(vals)...)
}

func ToArgGroup[T any](vals ...T) bob.Expression {
	return ArgGroup(internal.ToAnySlice(vals)...)
}
