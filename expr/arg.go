package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

func Arg(v any) query.Expression {
	return arg{val: v}
}

type arg struct {
	val any
}

func (a arg) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	d.WriteArg(w, start)
	return []any{a.val}, nil
}

func Placeholder(n uint) query.Expression {
	return Args(make([]any, n)...)
}

// Comma separated list of arguments
func Args(vals ...any) query.Expression {
	return args{vals: vals}
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
