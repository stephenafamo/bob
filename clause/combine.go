package clause

import (
	"context"
	"errors"
	"io"

	"github.com/stephenafamo/bob"
)

var ErrNoCombinationStrategy = errors.New("combination strategy must be set")

const (
	Union     = "UNION"
	Intersect = "INTERSECT"
	Except    = "EXCEPT"
)

type Combines struct {
	Queries []Combine
}

func (c *Combines) AppendCombine(combine Combine) {
	c.Queries = append(c.Queries, combine)
}

type Combine struct {
	Strategy string
	Query    bob.Query
	All      bool
}

func (s *Combine) SetCombine(c Combine) {
	*s = c
}

func (s Combine) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if s.Strategy == "" {
		return nil, ErrNoCombinationStrategy
	}

	w.WriteString(s.Strategy)

	if s.All {
		w.WriteString(" ALL ")
	} else {
		w.WriteString(" ")
	}

	w.WriteString("(")

	args, err := s.Query.WriteQuery(ctx, w, start)
	if err != nil {
		return nil, err
	}

	w.WriteString(")")

	return args, nil
}
