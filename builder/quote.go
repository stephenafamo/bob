package builder

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/query"
)

// quoted and joined... something like "users"."id"
func (e Builder[T, B]) Quote(aa ...string) T {
	var ss = make([]any, len(aa))
	for k, v := range aa {
		ss[k] = v
	}

	return e.X(quoted(ss))
}

// Quotes the base... Should only be used for raw strings
func (x Chain[T, B]) Quote() T {
	return X[T, B](quoted([]any{x.Base}))
}

// dquoted and joined... something like "users"."id"
type quoted []any

func (q quoted) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if len(q) == 0 {
		return nil, nil
	}

	// wrap in parenthesis and join with comma
	for k, a := range q {
		s, ok := a.(string)
		if !ok {
			return nil, fmt.Errorf("trying to quote non-string: %v", a)
		}
		if k != 0 {
			w.Write([]byte("."))
		}

		d.WriteQuoted(w, s)
	}

	return nil, nil
}

// single quoted raw string
func (e Builder[T, B]) S(s string) T {
	return e.X(rawString(s))
}

type rawString string

func (s rawString) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte("'"))
	w.Write([]byte(s))
	w.Write([]byte("'"))

	return nil, nil
}
