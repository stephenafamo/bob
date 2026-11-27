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

	Distinct bool
	clause.OrderBy
	w *clause.Window

	Alias   string
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

func (f *Function) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if f.name == "" {
		return nil, nil
	}

	w.WriteString(f.name)
	w.WriteString("(")

	if f.Distinct {
		w.WriteString("DISTINCT ")
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

	w.WriteString(")")

	if len(f.Columns) > 0 || len(f.Alias) > 0 {
		w.WriteString(" AS ")
	}

	if len(f.Alias) > 0 {
		w.WriteString(f.Alias)
		w.WriteString(" ")
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

func (c columnDef) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString(c.name + " " + c.dataType)

	return nil, nil
}
