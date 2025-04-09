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

func (s Combine) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if s.Strategy == "" {
		return nil, ErrNoCombinationStrategy
	}

	w.Write([]byte(s.Strategy))

	if s.All {
		w.Write([]byte(" ALL "))
	} else {
		w.Write([]byte(" "))
	}

	args, err := s.Query.WriteQuery(ctx, w, start)
	if err != nil {
		return nil, err
	}

	return args, nil
}
