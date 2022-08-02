package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type Select struct {
	Columns   []any
	Modifiers []any
}

func (s *Select) AppendSelect(columns ...any) {
	s.Columns = append(s.Columns, columns...)
}

func (s Select) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	w.Write([]byte("SELECT "))

	modArgs, err := bob.ExpressSlice(w, d, start+len(args), s.Modifiers, "", " ", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, modArgs...)

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
