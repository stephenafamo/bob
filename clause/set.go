package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type Set struct {
	Set []any
}

func (s *Set) AppendSet(exprs ...any) {
	s.Set = append(s.Set, exprs...)
}

func (s Set) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, s.Set, "", ",\n", "")
}
