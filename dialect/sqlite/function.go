package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/builder"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

type function struct {
	name string
	args []any

	// Used in value functions. Supported by Sqlite and Postgres
	filter []any

	// For chain methods
	builder.Chain[chain, chain]
}

// A function can be a target for a query
func (f *function) Apply(q *expr.FromItem) {
	q.Table = f
}

func (f *function) Filter(e ...any) *function {
	f.filter = append(f.filter, e...)

	return f
}

func (f *function) Over(window any) chain {
	return chain{Chain: builder.Chain[chain, chain]{
		Base: query.ExpressionFunc(func(w io.Writer, d query.Dialect, start int) ([]any, error) {
			largs, err := query.Express(w, d, start, f)
			if err != nil {
				return nil, err
			}

			rargs, err := query.ExpressIf(w, d, start+len(largs), window, true, "OVER (", ")")
			if err != nil {
				return nil, err
			}

			return append(largs, rargs...), nil
		}),
	}}
}

func (f function) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if f.name == "" {
		return nil, nil
	}

	w.Write([]byte(f.name))
	w.Write([]byte("("))
	args, err := query.ExpressSlice(w, d, start, f.args, "", ", ", "")
	if err != nil {
		return nil, err
	}
	w.Write([]byte(")"))

	filterArgs, err := query.ExpressSlice(w, d, start, f.filter, " FILTER (WHERE ", " AND ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, filterArgs...)

	return args, nil
}
