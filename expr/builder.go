package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

// Build an expression
func X(start any) Builder {
	return Builder{base: start}
}

// To be embeded in query mods
type ExpressionBuilder struct{}

func (e ExpressionBuilder) X(start any) Builder {
	return Builder{base: start}
}

type Builder struct {
	base any
}

func (x Builder) WriteSQL(w io.Writer, d query.Dialect, start int) (args []any, err error) {
	return query.Express(w, d, start, x.base)
}

// NOT
func (x Builder) NOT(exp any) query.Expression {
	return X(startEnd{prefix: "NOT ", expr: exp})
}

// IS DISTINCT FROM
func (x Builder) IS(exp any) Builder {
	return X(startEnd{expr: x.base, suffix: " IS DISTINCT FROM"})
}

// IS NOT DISTINCT FROM
func (x Builder) ISNOT() Builder {
	return X(startEnd{expr: x.base, suffix: " IS NOT DISTINCT FROM"})
}

// IS NUll
func (x Builder) ISNULL() Builder {
	return X(startEnd{expr: x.base, suffix: " NULL"})
}

// IS NOT NUll
func (x Builder) ISNOTNULL() Builder {
	return X(startEnd{expr: x.base, suffix: " IS NOT NULL"})
}

// Equal
func (x Builder) EQ(target any) Builder {
	return X(leftRight{left: x.base, right: target, operator: "="})
}

// Not Equal
func (x Builder) NE(target any) Builder {
	return X(leftRight{left: x.base, right: target, operator: "<>"})
}

// Less than
func (x Builder) LT(target any) Builder {
	return X(leftRight{left: x.base, right: target, operator: "<"})
}

// Less than or equal to
func (x Builder) LTE(target any) Builder {
	return X(leftRight{left: x.base, right: target, operator: "<="})
}

// Greater than
func (x Builder) GT(target any) Builder {
	return X(leftRight{left: x.base, right: target, operator: ">"})
}

// Greater than or equal to
func (x Builder) GTE(target any) Builder {
	return X(leftRight{left: x.base, right: target, operator: ">="})
}

// IN
func (x Builder) IN(vals ...any) Builder {
	return X(leftRight{left: x.base, right: Group(vals...), operator: "IN"})
}

// NOT IN
func (x Builder) NIN(vals ...any) Builder {
	return X(leftRight{left: x.base, right: Group(vals...), operator: "NOT IN"})
}

// OR
func (x Builder) OR(target any) Builder {
	return X(leftRight{operator: "OR", left: x.base, right: target})
}

// AND
func (x Builder) AND(target any) Builder {
	return X(leftRight{operator: "AND", left: x.base, right: target})
}

// Concatenate: `||``
func (x Builder) CONCAT(target any) Builder {
	return X(leftRight{operator: "||", left: x.base, right: target})
}

// Subtract
func (x Builder) MINUS(target any) Builder {
	return X(leftRight{operator: "-", left: x.base, right: target})
}

func (x Builder) AS(alias string) query.Expression {
	return leftRight{left: x.base, operator: " AS ", right: Quote(alias)}
}
