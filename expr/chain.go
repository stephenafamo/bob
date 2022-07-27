package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Chain[T any, B builder[T]] struct {
	Base any
}

// WriteSQL satisfies the query.Expression interface
func (x Chain[T, B]) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.Express(w, d, start, x.Base)
}

// IS DISTINCT FROM
func (x Chain[T, B]) Is(exp any) T {
	return X[T, B](Join{Exprs: []any{x.Base, "IS DISTINCT FROM", exp}})
}

// IS NOT DISTINCT FROM
func (x Chain[T, B]) IsNot(exp any) T {
	return X[T, B](Join{Exprs: []any{x.Base, "IS NOT DISTINCT FROM", exp}})
}

// IS NUll
func (x Chain[T, B]) IsNull() T {
	return X[T, B](Join{Exprs: []any{x.Base, "IS NULL"}})
}

// IS NOT NUll
func (x Chain[T, B]) IsNotNull() T {
	return X[T, B](Join{Exprs: []any{x.Base, "IS NOT NULL"}})
}

// Equal
func (x Chain[T, B]) EQ(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "="})
}

// Not Equal
func (x Chain[T, B]) NE(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<>"})
}

// Less than
func (x Chain[T, B]) LT(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<"})
}

// Less than or equal to
func (x Chain[T, B]) LTE(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<="})
}

// Greater than
func (x Chain[T, B]) GT(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: ">"})
}

// Greater than or equal to
func (x Chain[T, B]) GTE(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: ">="})
}

// IN
func (x Chain[T, B]) In(vals ...any) T {
	return X[T, B](leftRight{left: x.Base, right: group(vals), operator: "IN"})
}

// NOT IN
func (x Chain[T, B]) NotIn(vals ...any) T {
	return X[T, B](leftRight{left: x.Base, right: group(vals), operator: "NOT IN"})
}

// OR
func (x Chain[T, B]) Or(targets ...any) T {
	return X[T, B](Join{Exprs: append([]any{x.Base}, targets...), Sep: " OR "})
}

// AND
func (x Chain[T, B]) And(targets ...any) T {
	return X[T, B](Join{Exprs: append([]any{x.Base}, targets...), Sep: " AND "})
}

// Concatenate: `||``
func (x Chain[T, B]) Concat(targets ...any) T {
	return X[T, B](Join{Exprs: append([]any{x.Base}, targets...), Sep: " || "})
}

// BETWEEN a AND b
func (x Chain[T, B]) Between(a, b any) T {
	return X[T, B](Join{Exprs: []any{x.Base, "BETWEEN", a, "AND", b}})
}

// NOT BETWEEN a AND b
func (x Chain[T, B]) NotBetween(a, b any) T {
	return X[T, B](Join{Exprs: []any{
		x.Base, "NOT BETWEEN", a, "AND", b,
	}})
}

// Subtract
func (x Chain[T, B]) Minus(target any) T {
	return X[T, B](leftRight{operator: "-", left: x.Base, right: target})
}

// As does not return a new chain. Should be used at the end of an expression
// useful for columns
func (x Chain[T, B]) As(alias string) query.Expression {
	return leftRight{left: x.Base, operator: " AS ", right: quoted{alias}}
}
