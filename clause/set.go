package clause

import (
	"context"
	"errors"
	"io"

	"github.com/stephenafamo/bob"
)

var ErrEmptySetExpression = errors.New("SET clause must have at least one assignment expression")

type Set struct {
	Set []any
}

func (s *Set) AppendSet(exprs ...any) {
	s.Set = append(s.Set, exprs...)
}

func (s Set) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if len(s.Set) == 0 {
		return nil, ErrEmptySetExpression
	}
	return bob.ExpressSlice(ctx, w, d, start, s.Set, "", ",\n", "")
}
