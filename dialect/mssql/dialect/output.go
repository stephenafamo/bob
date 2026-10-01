package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// Output represents the T-SQL OUTPUT clause
// OUTPUT inserted.col1, deleted.col2
type Output struct {
	Expressions []any
}

func (o *Output) HasOutput() bool {
	return len(o.Expressions) > 0
}

func (o *Output) AppendOutput(exprs ...any) {
	o.Expressions = append(o.Expressions, exprs...)
}

func (o Output) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, o.Expressions, "OUTPUT ", ", ", "")
}
