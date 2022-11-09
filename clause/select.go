package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type SelectList struct {
	Columns []any
}

func (s *SelectList) AppendSelect(columns ...any) {
	s.Columns = append(s.Columns, columns...)
}

func (s SelectList) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	if len(s.Columns) > 0 {
		colArgs, err := bob.ExpressSlice(w, d, start+len(args), s.Columns, "", ", ", "")
		if err != nil {
			return nil, err
		}
		args = append(args, colArgs...)
	} else {
		w.Write([]byte("*"))
	}

	return args, nil
}
