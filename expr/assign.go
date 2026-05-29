package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// assignment is a column = value expression for SET / DO UPDATE SET clauses.
// Unlike comparisons built with EQ, it is not wrapped in parentheses.
type assignment struct {
	left  any
	right any
}

// ShouldOmitParens reports that SET assignments must not be parenthesized.
func (assignment) ShouldOmitParens() bool { return true }

func (a assignment) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return leftRight{operator: "=", left: a.left, right: a.right}.WriteSQL(ctx, w, d, start)
}

// Assign builds left = right for SET clauses (no surrounding parentheses).
func Assign(left, right any) bob.Expression {
	return assignment{left: left, right: right}
}
