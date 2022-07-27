package clause

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Limit struct {
	// Some DBs (e.g. SQite) can take an expression
	// It is up to the mods to enforce any extra conditions
	Count any
}

func (l *Limit) SetLimit(limit Limit) {
	*l = limit
}

func (l Limit) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressIf(w, d, start, l.Count, l.Count != nil, "LIMIT ", "")
}
