package expr

import (
	"io"

	"github.com/stephenafamo/bob"
)

func Quote(aa ...string) bob.Expression {
	return quoted(aa)
}

// quoted and joined... something like "users"."id"
type quoted []string

func (q quoted) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(q) == 0 {
		return nil, nil
	}

	// wrap in parenthesis and join with comma
	for k, a := range q {
		if k != 0 {
			w.Write([]byte("."))
		}

		d.WriteQuoted(w, a)
	}

	return nil, nil
}
