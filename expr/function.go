package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

func Func(name string, args ...any) Function {
	return Function{
		name: name,
		args: args,
	}
}

type Functions []any

func (f Functions) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if len(f) > 1 {
		w.Write([]byte("ROWS FROM ("))
	}

	args, err := query.ExpressSlice(w, d, start, f, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if len(f) > 1 {
		w.Write([]byte(")"))
	}

	return args, nil
}

type Function struct {
	name string
	args []any

	// Used in window functions. Supported by Sqlite and Postgres
	filter []any

	// Used when using as the source of a query
	alias   string
	columns []columnDef
}

type funcMod[Q interface{ AppendFunction(Function) }] Function

func (j funcMod[Q]) Apply(q Q) {
	q.AppendFunction(Function(j))
}

func (f Function) ToMod() funcMod[*FromItem] {
	return funcMod[*FromItem](f)
}

func (f Function) Filter(e ...any) Function {
	f.filter = append(f.filter, e...)

	return f
}

func (f Function) Alias(name string) Function {
	f.name = name
	return f
}

func (f Function) Col(name, datatype string) Function {
	f.columns = append(f.columns, columnDef{
		name:     name,
		dataType: datatype,
	})

	return f
}

func (f Function) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
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

	if len(f.columns) > 0 {
		w.Write([]byte(" AS "))
	}

	if len(f.alias) > 0 {
		w.Write([]byte(f.alias))
		w.Write([]byte(" "))
	}

	colArgs, err := query.ExpressSlice(w, d, start+len(args), f.columns, "(", ", ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, colArgs...)

	return args, nil
}

type columnDef struct {
	name     string
	dataType string
}

func (c columnDef) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte(c.name))
	w.Write([]byte(" "))
	w.Write([]byte(c.dataType))

	return nil, nil
}
