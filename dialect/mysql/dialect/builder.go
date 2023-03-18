package dialect

import (
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

//nolint:gochecknoglobals
var (
	iLike = expr.Raw("ILIKE")
)

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
	x.WriteSQL(&w, Dialect, 1) //nolint:errcheck
	return w.String()
}

// ILIKE val
func (x Expression) ILike(val bob.Expression) Expression {
	return expr.X[Expression, Expression](expr.Join{Exprs: []bob.Expression{
		x.Base, iLike, val,
	}})
}
