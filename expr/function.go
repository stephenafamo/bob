package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

func Func(name string, args ...any) function {
	return function{
		name: name,
		args: args,
	}
}

type function struct {
	name string
	args []any

	// Used in window functions. Supported by Sqlite and Postgres
	filter []any

	// Used when using as the source of a query
	alias   string
	columns []columnDef
}

func (f function) Filter(e ...any) function {
	f.filter = append(f.filter, e...)

	return f
}

func (f function) Alias(name string) function {
	f.name = name
	return f
}

func (f function) Col(name, datatype string) function {
	f.columns = append(f.columns, columnDef{
		name:     name,
		dataType: datatype,
	})

	return f
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
