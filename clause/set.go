package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Set struct {
	Set []any
}

func (s *Set) AppendSet(exprs ...any) {
	s.Set = append(s.Set, exprs...)
}

func (s Set) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, s.Set, "", ",\n", "")
}
