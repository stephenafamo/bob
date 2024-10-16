package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Offset struct {
	// Some DBs (e.g. SQite) can take an expression
	// It is up to the mods to enforce any extra conditions
	Count any
}

func (o *Offset) SetOffset(offset any) {
	o.Count = offset
}

func (o Offset) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressIf(ctx, w, d, start, o.Count, o.Count != nil, "OFFSET ", "")
}
