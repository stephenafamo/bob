package expr

import (
	"io"

	"github.com/stephenafamo/bob"
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

func (a args) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(a.vals) == 0 {
		return nil, nil
	}

	if a.grouped {
		w.Write([]byte(openPar))
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
