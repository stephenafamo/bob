package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Set struct {
	Set []any
}

func (s *Set) AppendSet(exprs ...any) {
	s.Set = append(s.Set, exprs...)
}

func (s Set) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, s.Set, "SET\n", ",\n", "")
}
