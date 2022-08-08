package mysql

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

	// For chain methods
	expr.Chain[chain, chain]
}

// A function can be a target for a query
func (f *function) Apply(q *clause.From) {
	q.Table = f
}

func (f *function) Filter(e ...any) *function {
	f.filter = append(f.filter, e...)

	return f
}

func (f *function) Over(window string) *functionOver {
	fo := &functionOver{
		function: f,
	}
	fo.def = fo
	fo.Base = fo
	return fo
}

func (f function) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
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

	return args, nil
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
