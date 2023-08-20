package bob

import (
	"fmt"
	"io"
)

func Cache(q Query) (BaseQuery[*cached], error) {
	return CacheN(q, 1)
}

func CacheN(q Query, start int) (BaseQuery[*cached], error) {
	query, args, err := BuildN(q, start)
	if err != nil {
		return BaseQuery[*cached]{}, err
	}

	return BaseQuery[*cached]{
		Expression: &cached{
			query: []byte(query),
			args:  args,
			start: start,
		},
	}, nil
}

type WrongStartError struct {
	Expected int
	Got      int
}

func (e WrongStartError) Error() string {
	return fmt.Sprintf("expected to start at %d, started at %d", e.Expected, e.Got)
}

type cached struct {
	query []byte
	args  []any
	start int
}

// WriteSQL implements Expression.
func (c *cached) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
	if start != c.start {
		return nil, WrongStartError{Expected: c.start, Got: start}
	}

	if _, err := w.Write(c.query); err != nil {
		return nil, err
	}

	return c.args, nil
}
