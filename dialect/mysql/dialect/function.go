package dialect

import (
	"context"
	"io"

	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/clause"
	"github.com/twitter-payments/bob/expr"
)

func NewFunction(name string, args ...any) *Function {
	f := &Function{name: name, args: args}
	f.Chain = expr.Chain[Expression, Expression]{Base: f}

	return f
}

type Function struct {
	name string
	args []any

	// Used in value functions. Supported by Sqlite and Postgres
	Distinct bool
	clause.OrderBy
	Filter []any
	w      *clause.Window

	// For chain methods
	expr.Chain[Expression, Expression]
}

func (f *Function) SetWindow(w clause.Window) {
	f.w = &w
}

func (f Function) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if f.name == "" {
		return nil, nil
	}

	w.Write([]byte(f.name))
	w.Write([]byte("("))

	if f.Distinct {
		w.Write([]byte("DISTINCT "))
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, f.args, "", ", ", "")
	if err != nil {
		return nil, err
	}

	orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), f.OrderBy,
		len(f.OrderBy.Expressions) > 0, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	w.Write([]byte(")"))

	filterArgs, err := bob.ExpressSlice(ctx, w, d, start, f.Filter, " FILTER (WHERE ", " AND ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, filterArgs...)

	winargs, err := bob.ExpressIf(ctx, w, d, start+len(args), f.w, f.w != nil, "OVER (", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, winargs...)

	return args, nil
}
