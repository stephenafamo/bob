package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type SelectList struct {
	Columns []any
	// necessary to be able to treat preloaders
	// like any other query Mod
	PreloadColumns []any
}

func (s *SelectList) CountSelectCols() int {
	return len(s.Columns)
}

func (s *SelectList) SetSelect(columns ...any) {
	s.Columns = columns
}

func (s *SelectList) SetPreloadSelect(columns ...any) {
	s.PreloadColumns = columns
}

func (s *SelectList) AppendSelect(columns ...any) {
	s.Columns = append(s.Columns, columns...)
}

func (s *SelectList) AppendPreloadSelect(columns ...any) {
	s.PreloadColumns = append(s.PreloadColumns, columns...)
}

func (s SelectList) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	all := append(s.Columns, s.PreloadColumns...)
	if len(all) > 0 {
		colArgs, err := bob.ExpressSlice(w, d, start+len(args), all, "", ", ", "")
		if err != nil {
			return nil, err
		}
		args = append(args, colArgs...)
	} else {
		w.Write([]byte("*"))
	}

	return args, nil
}
