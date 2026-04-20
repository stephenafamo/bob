package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

func Quote(aa ...string) bob.Expression {
	allNonEmpty := true
	for _, v := range aa {
		if v == "" {
			allNonEmpty = false
			break
		}
	}

	if allNonEmpty {
		return quoted(aa)
	}

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

func (q quoted) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return nil, q.WriteSQLTo(ctx, w, d, start, nil)
}

func (q quoted) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	if len(q) == 0 {
		return nil
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

	return nil
}
