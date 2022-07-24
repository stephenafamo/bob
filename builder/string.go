package builder

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type rawString string

func (s rawString) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte("'"))
	w.Write([]byte(s))
	w.Write([]byte("'"))

	return nil, nil
}
