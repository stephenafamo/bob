package psql

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

type function struct {
	name string
	args []any

	// Used in value functions. Supported by Sqlite and Postgres
	filter []any

	alias   string // used with there should be an alias before the columns
	columns []columnDef

	// For chain methods
	expr.Chain[chain, chain]
}

func (f *function) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
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

	if len(f.columns) > 0 {
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

// Multiple functions can be uses as a goup with ROWS FROM
func (f *function) Apply(q *clause.FromItem) {
	switch fs := q.Table.(type) {
	case functions:
		q.Table = append(fs, f)
	default:
		q.Table = functions{f}
	}
}

func (f *function) Over(window string) *functionOver {
	fo := &functionOver{
		function: f,
	}
	fo.def = fo
	fo.Base = fo
	return fo
}

func (f *function) As(alias string) *function {
	f.alias = alias
	return f
}

func (f *function) Col(name, datatype string) *function {
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
	w.Write([]byte(c.name))
	w.Write([]byte(" "))
	w.Write([]byte(c.dataType))

	return nil, nil
}

type functionOver struct {
	function *function
	clause.WindowDef
	windowChain[*functionOver]
	expr.Chain[chain, chain]
}

func (wr *functionOver) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	fargs, err := bob.Express(w, d, start, wr.function)
	if err != nil {
		return nil, err
	}

	winargs, err := bob.ExpressIf(w, d, start+len(fargs), wr.WindowDef, true, "OVER (", ")")
	if err != nil {
		return nil, err
	}

	return append(fargs, winargs...), nil
}

type functions []any

func (f functions) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
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
