package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

func NewFunction(name string, args ...any) *Function {
	f := &Function{name: name, args: args}
	f.Chain = expr.Chain[Expression, Expression]{Base: f}

	return f
}

type Function struct {
	name string
	args []any

	Distinct    bool
	WithinGroup bool
	clause.OrderBy
	Filter []any
	w      *clause.Window

	Alias   string // used when there should be an alias before the columns
	Columns []columnDef

	expr.Chain[Expression, Expression]
}

func (f *Function) SetWindow(w clause.Window) {
	f.w = &w
}

func (f *Function) AppendColumn(name, datatype string) {
	f.Columns = append(f.Columns, columnDef{
		name:     name,
		dataType: datatype,
	})
}

func (f *Function) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
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

	if !f.WithinGroup {
		orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), f.OrderBy,
			len(f.OrderBy.Expressions) > 0, " ", "")
		if err != nil {
			return nil, err
		}
		args = append(args, orderArgs...)
	}
	w.Write([]byte(")"))

	if f.WithinGroup {
		orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), f.OrderBy,
			len(f.OrderBy.Expressions) > 0, " WITHIN GROUP (", ")")
		if err != nil {
			return nil, err
		}
		args = append(args, orderArgs...)
	}

	filterArgs, err := bob.ExpressSlice(ctx, w, d, start, f.Filter, " FILTER (WHERE ", " AND ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, filterArgs...)

	if len(f.Columns) > 0 || len(f.Alias) > 0 {
		w.Write([]byte(" AS "))
	}

	if len(f.Alias) > 0 {
		w.Write([]byte(f.Alias))
		w.Write([]byte(" "))
	}

	colArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), f.Columns, "(", ", ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, colArgs...)

	winargs, err := bob.ExpressIf(ctx, w, d, start+len(args), f.w, f.w != nil, "OVER (", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, winargs...)

	return args, nil
}

type columnDef struct {
	name     string
	dataType string
}

func (c columnDef) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte(c.name + " " + c.dataType))

	return nil, nil
}

type Functions []*Function

func (f Functions) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(f) > 1 {
		w.Write([]byte("ROWS FROM ("))
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, f, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if len(f) > 1 {
		w.Write([]byte(")"))
	}

	return args, nil
}
