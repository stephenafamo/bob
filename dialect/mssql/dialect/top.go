package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// Top represents T-SQL's TOP clause
// SELECT TOP (n) [PERCENT] [WITH TIES] ...
type Top struct {
	Count   any
	Percent bool
	// WithTies requires ORDER BY
	WithTies bool
}

func (t *Top) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if t == nil || t.Count == nil {
		return nil, nil
	}

	args, err := bob.ExpressIf(ctx, w, d, start, t.Count, true, "TOP (", ")")
	if err != nil {
		return nil, err
	}

	if t.Percent {
		w.WriteString(" PERCENT")
	}

	if t.WithTies {
		w.WriteString(" WITH TIES")
	}

	return args, nil
}
