package clause

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Offset struct {
	// Some DBs (e.g. SQite) can take an expression
	// It is up to the mods to enforce any extra conditions
	Count any
}

func (o *Offset) SetOffset(offset Offset) {
	*o = offset
}

func (o Offset) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressIf(w, d, start, o.Count, o.Count != nil, "OFFSET ", "")
}
