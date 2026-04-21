package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type defaultReturning struct {
	expr bob.Expression
}

func DefaultReturning(expr bob.Expression) bob.Expression {
	return defaultReturning{expr: expr}
}

func (d defaultReturning) IsDefaultReturning() bool {
	return true
}

func (d defaultReturning) WriteSQL(ctx context.Context, w io.StringWriter, dl bob.Dialect, start int) ([]any, error) {
	return d.expr.WriteSQL(ctx, w, dl, start)
}
