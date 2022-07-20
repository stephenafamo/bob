package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Select struct {
	Columns  []any
	Distinct Distinct
}

type Distinct struct {
	Distinct bool
	On       []any
}

func (s *Select) AppendSelect(columns ...any) {
	s.Columns = append(s.Columns, columns...)
}

func (s *Select) SetDistinct(distinct Distinct) {
	s.Distinct = distinct
}

func (s Select) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	w.Write([]byte("SELECT "))

	if s.Distinct.Distinct {
		w.Write([]byte("DISTINCT "))
		onArgs, err := query.ExpressSlice(w, d, start+len(args), s.Distinct.On, "ON (", ", ", ") ")
		if err != nil {
			return nil, err
		}

		args = append(args, onArgs...)
	}

	if len(s.Columns) > 0 {
		colArgs, err := query.ExpressSlice(w, d, start+len(args), s.Columns, "", ", ", "")
		if err != nil {
			return nil, err
		}
		args = append(args, colArgs...)
	} else {
		w.Write([]byte("*"))
	}

	return args, nil
}
