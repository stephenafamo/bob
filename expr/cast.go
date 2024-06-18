package expr

import (
	"io"

	"github.com/stephenafamo/bob"
)

func Cast(e bob.Expression, typname string) bob.Expression {
	return cast{e: e, typname: typname}
}

type cast struct {
	e       bob.Expression
	typname string
}

func (c cast) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressIf(w, d, start, c.e, c.e != nil, "CAST(", " AS "+c.typname+")")
}
