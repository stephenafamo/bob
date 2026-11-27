package dialect

import (
	"context"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

func NewExpression(exp bob.Expression) Expression {
	return Expression{}.New(exp)
}

type Expression struct {
	expr.Chain[Expression, Expression]
}

func (Expression) New(exp bob.Expression) Expression {
	var b Expression
	b.Base = exp
	return b
}

// Implements fmt.Stringer()
func (x Expression) String() string {
	w := strings.Builder{}
	x.WriteSQL(context.Background(), &w, Dialect, 1) //nolint:errcheck
	return w.String()
}
