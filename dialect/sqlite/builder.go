package sqlite

import (
	"strings"

	"github.com/stephenafamo/bob/expr"
)

type Expression struct {
	expr.Chain[Expression, Expression]
}

func (Expression) New(exp any) Expression {
	var b Expression
	b.Base = exp
	return b
}

// Implements fmt.Stringer()
func (x Expression) String() string {
	w := strings.Builder{}
	x.WriteSQL(&w, dialect, 1) //nolint:errcheck
	return w.String()
}
