package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

// Build an expression
func X(exp any) Builder {
	// Wrap in parenthesis if not a raw string or string in quotes
	switch exp.(type) {
	case string, rawString:
		break
	default:
		exp = P(exp)
	}

	return Builder{base: exp}
}

// prefix the expression with a NOT
func NotX(exp any) Builder {
	return Builder{base: P(startEnd{prefix: "NOT ", expr: X(exp)})}
}

// To be embeded in query mods
type ExpressionBuilder struct{}

// Start building an expression
func (e ExpressionBuilder) X(exp any) Builder {
	return X(exp)
}

// prefix the expression with a NOT
func (e ExpressionBuilder) NotX(exp any) Builder {
	return NotX(exp)
}

type Builder struct {
	base any
}

// WriteSQL satisfies the query.Expression interface
func (x Builder) WriteSQL(w io.Writer, d query.Dialect, start int) (args []any, err error) {
	return query.Express(w, d, start, x.base)
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

// As does not return a Builder. Should be used at the end of an expression
func (x Builder) AS(alias string) query.Expression {
	return leftRight{left: x.base, operator: " AS ", right: Quote(alias)}
}
