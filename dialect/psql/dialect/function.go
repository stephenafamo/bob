package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

func NewFunction(name any, args ...any) *Function {
	f := &Function{name: name, args: args}
	f.Chain = expr.Chain[Expression, Expression]{Base: f}

	return f
}

type Function struct {
	name any
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

func (f *Function) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	nameArgs, err := bob.Express(ctx, w, d, start, f.name)
	if err != nil {
		return nil, err
	}

	w.WriteString("(")
	start += len(nameArgs)

	if f.Distinct {
		w.WriteString("DISTINCT ")
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
	w.WriteString(")")

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
		w.WriteString(" AS ")
	}

	if len(f.Alias) > 0 {
		d.WriteQuoted(w, f.Alias)
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

	return append(nameArgs, args...), nil
}

type columnDef struct {
	name     string
	dataType string
}

func (c columnDef) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	d.WriteQuoted(w, c.name)
	w.WriteString(" ")
	w.WriteString(c.dataType)

	return nil, nil
}

// Functions renders ROWS FROM (f1, f2, ...) for multiple table functions in one
// from_item.
type Functions []*Function

// TableFunctions returns a FROM/USING expression for table functions: a single
// function_name(...) or ROWS FROM (...) when multiple are given.
func TableFunctions(funcs ...*Function) bob.Expression {
	switch len(funcs) {
	case 0:
		return nil
	case 1:
		return funcs[0]
	default:
		return Functions(funcs)
	}
}

func (f Functions) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if len(f) > 1 {
		w.WriteString("ROWS FROM (")
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, f, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if len(f) > 1 {
		w.WriteString(")")
	}

	return args, nil
}
