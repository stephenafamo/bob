package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type Limit struct {
	// Some DBs (e.g. SQite) can take an expression
	// It is up to the mods to enforce any extra conditions
	Count any
}

func (l *Limit) SetLimit(limit any) {
	l.Count = limit
}

func (l Limit) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressIf(w, d, start, l.Count, l.Count != nil, "LIMIT ", "")
}
