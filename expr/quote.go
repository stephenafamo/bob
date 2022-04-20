package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

func Quote(ss ...string) query.Expression {
	return quoted(ss)
}

// double quoted and joined... something like "users"."id"
type quoted []string

func (q quoted) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if len(q) == 0 {
		return nil, nil
	}

	// wrap in parenthesis and join with comma
	for k, s := range q {
		if k != 0 {
			w.Write([]byte("."))
		}

		d.WriteQuoted(w, s)
	}

	return nil, nil
}

// single quoted raw string
func S(s string) query.Expression {
	return rawString(s)
}

type rawString string

func (s rawString) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte("'"))
	w.Write([]byte(s))
	w.Write([]byte("'"))

	return nil, nil
}
