package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

// Comma separated list of arguments
func Arg(vals ...any) query.Expression {
	return args{vals: vals}
}

func Placeholder(n uint) query.Expression {
	return Arg(make([]any, n)...)
}

type arg struct {
	val any
}

func (a arg) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	d.WriteArg(w, start)
	return []any{a.val}, nil
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
