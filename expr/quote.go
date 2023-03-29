package expr

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
)

func Quote(aa ...string) bob.Expression {
	ss := make([]any, 0, len(aa))
	for _, v := range aa {
		if v == "" {
			continue
		}
		ss = append(ss, v)
	}

	return quoted(ss)
}

// quoted and joined... something like "users"."id"
type quoted []any

func (q quoted) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
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
