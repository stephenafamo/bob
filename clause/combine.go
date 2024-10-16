package clause

import (
	"context"
	"errors"
	"io"

	"github.com/stephenafamo/bob"
)

var ErrNoCombinationStrategy = errors.New("Combination strategy must be set")

const (
	Union     = "UNION"
	Intersect = "INTERSECT"
	Except    = "EXCEPT"
)

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

	args, err := bob.Express(ctx, w, d, start, s.Query)
	if err != nil {
		return nil, err
	}

	return args, nil
}
