package dialect

import (
	"context"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

type Expression struct {
	expr.Chain[Expression, Expression]
}

func (Expression) New(exp bob.Expression) Expression {
	var b Expression
	b.Base = exp
	return b
}

// SQL: CONCAT(x, a, b)
func (x Expression) Concat(targets ...bob.Expression) Expression {
	return Concat(append([]bob.Expression{x.Base}, targets...)...)
}

// SQL: CONCAT(a, b, c)
func Concat(exps ...bob.Expression) Expression {
	args := make([]any, len(exps))
	for i, exp := range exps {
		args[i] = exp
	}

	return expr.X[Expression, Expression](NewFunction("CONCAT", args...))
}

// Implements fmt.Stringer()
func (x Expression) String() string {
	w := strings.Builder{}
	x.WriteSQL(context.Background(), &w, Dialect, 1) //nolint:errcheck
	return w.String()
}
