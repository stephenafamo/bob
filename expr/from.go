package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type From struct {
	Tables []any
	Joins  []Join
}

func (f *From) AppendFrom(table Table) {
	f.Tables = append(f.Tables, table)
}

func (f *From) AppendJoin(j Join) {
	f.Joins = append(f.Joins, j)
}

func (f From) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if len(f.Tables) == 0 {
		return nil, nil
	}

	w.Write([]byte("FROM "))

	args, err := query.ExpressSlice(w, d, start, f.Tables, "", ", ", " ")
	if err != nil {
		return nil, err
	}

	joinArgs, err := query.ExpressSlice(w, d, start+len(args), f.Joins, "\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, joinArgs...)

	return args, nil
}
