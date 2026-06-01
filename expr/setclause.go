package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// assignment is a column = value expression used when rendering SET clauses.
// It is not wrapped in parentheses (see prepareSetAssignment).
type assignment struct {
	left  any
	right any
}

// ShouldOmitParens reports that SET assignments must not be parenthesized.
func (assignment) ShouldOmitParens() bool { return true }

func (a assignment) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return leftRight{operator: "=", left: a.left, right: a.right}.WriteSQL(ctx, w, d, start)
}

// ExpressionBase returns the inner expression of a chain wrapper.
func (x Chain[T, B]) ExpressionBase() bob.Expression {
	return x.Base
}

type expressionBase interface {
	ExpressionBase() bob.Expression
}

// prepareSetAssignment rewrites e for SET / ON CONFLICT DO UPDATE SET rendering.
// Equalities built with EQ lose the comparison parentheses; other expressions are unchanged.
func prepareSetAssignment(e any) any {
	inner := e
	for {
		b, ok := inner.(expressionBase)
		if !ok {
			break
		}
		base := b.ExpressionBase()
		if base == nil {
			break
		}
		inner = base
	}
	if g, ok := inner.(group); ok && len(g) == 1 {
		if lr, ok := g[0].(leftRight); ok && lr.operator == "=" {
			return assignment{left: lr.left, right: lr.right}
		}
	}
	if _, ok := inner.(assignment); ok {
		return inner
	}
	return e
}

// PrepareSetAssignments rewrites each expression for SET clause rendering.
func PrepareSetAssignments(items []any) []any {
	if len(items) == 0 {
		return items
	}
	out := make([]any, len(items))
	for i, e := range items {
		out[i] = prepareSetAssignment(e)
	}
	return out
}
