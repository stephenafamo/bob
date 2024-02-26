package dialect

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

func NewFunction(name string, args ...any) Function {
	return Function{name: name, args: args}
}

type Function struct {
	name string
	args []any

	// Used in value functions. Supported by Sqlite and Postgres
	filter []any

	alias   string // used when there should be an alias before the columns
	columns []columnDef

	// For chain methods
	expr.Chain[Expression, Expression]
}

func (f *Function) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if f.name == "" {
		return nil, nil
	}

	w.Write([]byte(f.name))
	w.Write([]byte("("))
	args, err := bob.ExpressSlice(w, d, start, f.args, "", ", ", "")
	if err != nil {
		return nil, err
	}
	w.Write([]byte(")"))

	filterArgs, err := bob.ExpressSlice(w, d, start, f.filter, " FILTER (WHERE ", " AND ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, filterArgs...)

	if len(f.columns) > 0 || len(f.alias) > 0 {
		w.Write([]byte(" AS "))
	}

	if len(f.alias) > 0 {
		w.Write([]byte(f.alias))
		w.Write([]byte(" "))
	}

	colArgs, err := bob.ExpressSlice(w, d, start+len(args), f.columns, "(", ", ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, colArgs...)

	return args, nil
}

func (f *Function) FilterWhere(e ...any) *functionOver {
	f.filter = append(f.filter, e...)

	fo := &functionOver{
		function: f,
	}
	fo.WindowChain = &WindowChain[*functionOver]{Wrap: fo}
	fo.Base = fo
	return fo
}

func (f *Function) Over() *functionOver {
	fo := &functionOver{
		function: f,
	}
	fo.WindowChain = &WindowChain[*functionOver]{Wrap: fo}
	fo.Base = fo
	return fo
}

func (f *Function) As(alias string) *Function {
	f.alias = alias
	return f
}

func (f *Function) Col(name, datatype string) *Function {
	f.columns = append(f.columns, columnDef{
		name:     name,
		dataType: datatype,
	})

	return f
}

type columnDef struct {
	name     string
	dataType string
}

func (c columnDef) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte(c.name + " " + c.dataType))

	return nil, nil
}

type functionOver struct {
	function *Function
	*WindowChain[*functionOver]
	expr.Chain[Expression, Expression]
}

func (wr *functionOver) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	fargs, err := bob.Express(w, d, start, wr.function)
	if err != nil {
		return nil, err
	}

	winargs, err := bob.ExpressIf(w, d, start+len(fargs), wr.def, true, "OVER (", ")")
	if err != nil {
		return nil, err
	}

	return append(fargs, winargs...), nil
}

type Functions []*Function

func (f Functions) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(f) > 1 {
		w.Write([]byte("ROWS FROM ("))
	}

	args, err := bob.ExpressSlice(w, d, start, f, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if len(f) > 1 {
		w.Write([]byte(")"))
	}

	return args, nil
}
