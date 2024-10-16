package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Chain[T bob.Expression, B builder[T]] struct {
	Base bob.Expression
}

// WriteSQL satisfies the bob.Expression interface
func (x Chain[T, B]) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.Express(ctx, w, d, start, x.Base)
}

// IS DISTINCT FROM
func (x Chain[T, B]) IsDistinctFrom(exp bob.Expression) T {
	return X[T, B](Join{Exprs: []bob.Expression{x.Base, isDistinctFrom, exp}})
}

// IS NOT DISTINCT FROM
func (x Chain[T, B]) IsNotDistinctFrom(exp bob.Expression) T {
	return X[T, B](Join{Exprs: []bob.Expression{x.Base, isNotDistinctFrom, exp}})
}

// IS NUll
func (x Chain[T, B]) IsNull() T {
	return X[T, B](Join{Exprs: []bob.Expression{x.Base, isNull}})
}

// IS NOT NUll
func (x Chain[T, B]) IsNotNull() T {
	return X[T, B](Join{Exprs: []bob.Expression{x.Base, isNotNull}})
}

// Generic Operator
func (x Chain[T, B]) OP(op string, target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: op})
}

// Equal
func (x Chain[T, B]) EQ(target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "="})
}

// Not Equal
func (x Chain[T, B]) NE(target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<>"})
}

// Less than
func (x Chain[T, B]) LT(target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<"})
}

// Less than or equal to
func (x Chain[T, B]) LTE(target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<="})
}

// Greater than
func (x Chain[T, B]) GT(target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: ">"})
}

// Greater than or equal to
func (x Chain[T, B]) GTE(target bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: ">="})
}

// IN
func (x Chain[T, B]) In(vals ...bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: group(vals), operator: "IN"})
}

// NOT IN
func (x Chain[T, B]) NotIn(vals ...bob.Expression) T {
	return X[T, B](leftRight{left: x.Base, right: group(vals), operator: "NOT IN"})
}

// OR
func (x Chain[T, B]) Or(targets ...bob.Expression) T {
	return X[T, B](Join{Exprs: append([]bob.Expression{x.Base}, targets...), Sep: " OR "})
}

// AND
func (x Chain[T, B]) And(targets ...bob.Expression) T {
	return X[T, B](Join{Exprs: append([]bob.Expression{x.Base}, targets...), Sep: " AND "})
}

// Concatenate: ||
func (x Chain[T, B]) Concat(targets ...bob.Expression) T {
	return X[T, B](Join{Exprs: append([]bob.Expression{x.Base}, targets...), Sep: " || "})
}

// BETWEEN a AND b
func (x Chain[T, B]) Between(a, b bob.Expression) T {
	return X[T, B](Join{Exprs: []bob.Expression{x.Base, between, a, and, b}})
}

// NOT BETWEEN a AND b
func (x Chain[T, B]) NotBetween(a, b bob.Expression) T {
	return X[T, B](Join{Exprs: []bob.Expression{
		x.Base, notBetween, a, and, b,
	}})
}

// Subtract
func (x Chain[T, B]) Minus(target bob.Expression) T {
	return X[T, B](leftRight{operator: "-", left: x.Base, right: target})
}

// Like operator
func (x Chain[T, B]) Like(target bob.Expression) T {
	return X[T, B](leftRight{operator: "LIKE", left: x.Base, right: target})
}

// As does not return a new chain. Should be used at the end of an expression
// useful for columns
func (x Chain[T, B]) As(alias string) bob.Expression {
	return leftRight{left: x.Base, operator: "AS", right: quoted{alias}}
}
