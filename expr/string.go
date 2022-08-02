package expr

import (
	"io"

	"github.com/stephenafamo/bob"
)

type rawString string

func (s rawString) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte("'"))
	w.Write([]byte(s))
	w.Write([]byte("'"))

	return nil, nil
}
