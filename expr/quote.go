package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

func Quote(aa ...string) bob.Expression {
	ss := make([]string, 0, len(aa))
	for _, v := range aa {
		if v == "" {
			continue
		}

		ss = append(ss, v)
	}

	return quoted(ss)
}

// quoted and joined... something like "users"."id"
type quoted []string

// Unqualified returns a quoted expression containing only the last identifier part.
func (q quoted) Unqualified() quoted {
	if len(q) == 0 {
		return quoted{}
	}

	return quoted{q[len(q)-1]}
}

// UnquotedLast returns the last element of the quoted slice, or empty string if empty.
func (q quoted) UnquotedLast() string {
	if len(q) == 0 {
		return ""
	}

	return q[len(q)-1]
}

func (q quoted) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if len(q) == 0 {
		return nil, nil
	}

	// wrap in parenthesis and join with comma
	k := 0 // not using the loop index to avoid empty strings
	for _, a := range q {
		if a == "" {
			continue
		}

		if k != 0 {
			w.WriteString(".")
		}
		k++

		d.WriteQuoted(w, a)
	}

	return nil, nil
}
