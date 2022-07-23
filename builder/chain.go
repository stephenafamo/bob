package builder

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
func (x Chain[T, B]) IS(exp any) T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " IS DISTINCT FROM"})
}

// IS NOT DISTINCT FROM
func (x Chain[T, B]) ISNOT() T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " IS NOT DISTINCT FROM"})
}

// IS NUll
func (x Chain[T, B]) ISNULL() T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " NULL"})
}

// IS NOT NUll
func (x Chain[T, B]) ISNOTNULL() T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " IS NOT NULL"})
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
func (x Chain[T, B]) IN(vals ...any) T {
	return X[T, B](leftRight{left: x.Base, right: group(vals), operator: "IN"})
}

// NOT IN
func (x Chain[T, B]) NIN(vals ...any) T {
	return X[T, B](leftRight{left: x.Base, right: group(vals), operator: "NOT IN"})
}

// OR
func (x Chain[T, B]) OR(target any) T {
	return X[T, B](leftRight{operator: "OR", left: x.Base, right: target})
}

// AND
func (x Chain[T, B]) AND(target any) T {
	return X[T, B](leftRight{operator: "AND", left: x.Base, right: target})
}

// Concatenate: `||``
func (x Chain[T, B]) CONCAT(target any) T {
	return X[T, B](leftRight{operator: "||", left: x.Base, right: target})
}

// Subtract
func (x Chain[T, B]) MINUS(target any) T {
	return X[T, B](leftRight{operator: "-", left: x.Base, right: target})
}

// As does not return a Builder. Should be used at the end of an expression
func (x Chain[T, B]) AS(alias string) query.Expression {
	var aliasExp = Chain[T, B]{Base: alias}.Quote()

	return leftRight{left: x.Base, operator: " AS ", right: aliasExp}
}
