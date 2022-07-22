package expr

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/query"
)

type builder[B any] interface {
	New(any) B
}

// Build an expression
func X[T any, B builder[T]](exp any) T {
	// Wrap in parenthesis if not a raw string or string in quotes
	switch exp.(type) {
	case string, rawString, args:
		break
	default:
		exp = P(exp)
	}

	var b B
	return b.New(exp)
}

// prefix the expression with a NOT
func NotX[T any, B builder[T]](exp any) T {
	var b B
	return b.New(P(StartEnd{prefix: "NOT ", expr: X[T, B](exp)}))
}

// To be embeded in query mods
type ExpressionBuilder[T any, B builder[T]] struct{}

// Start building an expression
func (e ExpressionBuilder[T, B]) X(exp any) T {
	return X[T, B](exp)
}

// prefix the expression with a NOT
func (e ExpressionBuilder[T, B]) NotX(exp any) T {
	return NotX[T, B](exp)
}

// Or
func (e ExpressionBuilder[T, B]) Or(args ...any) T {
	return e.X(sliceJoin{expr: args, operator: " OR "})
}

// And
func (e ExpressionBuilder[T, B]) And(args ...any) T {
	return e.X(sliceJoin{expr: args, operator: " AND "})
}

// Concatenation `||` operator
func (e ExpressionBuilder[T, B]) CONCAT(ss ...any) T {
	return e.X(sliceJoin{expr: ss, operator: " || "})
}

// Comma separated list of arguments
func (e ExpressionBuilder[T, B]) Arg(vals ...any) T {
	return e.X(args{vals: vals})
}

func (e ExpressionBuilder[T, B]) Placeholder(n uint) T {
	return e.Arg(make([]any, n)...)
}

// OVER: For window functions
func (e ExpressionBuilder[T, B]) OVER(f Function, window any) T {
	return e.X(query.ExpressionFunc(func(w io.Writer, d query.Dialect, start int) ([]any, error) {
		largs, err := query.Express(w, d, start, f)
		if err != nil {
			return nil, err
		}

		fmt.Fprint(w, " OVER (")

		rargs, err := query.Express(w, d, start+len(largs), window)
		if err != nil {
			return nil, err
		}

		fmt.Fprint(w, ")")

		return append(largs, rargs...), nil
	}))
}

type Common[T any, B builder[T]] struct {
	Base any
}

// WriteSQL satisfies the query.Expression interface
func (x Common[T, B]) WriteSQL(w io.Writer, d query.Dialect, start int) (args []any, err error) {
	return query.Express(w, d, start, x.Base)
}

// IS DISTINCT FROM
func (x Common[T, B]) IS(exp any) T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " IS DISTINCT FROM"})
}

// IS NOT DISTINCT FROM
func (x Common[T, B]) ISNOT() T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " IS NOT DISTINCT FROM"})
}

// IS NUll
func (x Common[T, B]) ISNULL() T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " NULL"})
}

// IS NOT NUll
func (x Common[T, B]) ISNOTNULL() T {
	return X[T, B](StartEnd{expr: x.Base, suffix: " IS NOT NULL"})
}

// Equal
func (x Common[T, B]) EQ(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "="})
}

// Not Equal
func (x Common[T, B]) NE(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<>"})
}

// Less than
func (x Common[T, B]) LT(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<"})
}

// Less than or equal to
func (x Common[T, B]) LTE(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: "<="})
}

// Greater than
func (x Common[T, B]) GT(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: ">"})
}

// Greater than or equal to
func (x Common[T, B]) GTE(target any) T {
	return X[T, B](leftRight{left: x.Base, right: target, operator: ">="})
}

// IN
func (x Common[T, B]) IN(vals ...any) T {
	return X[T, B](leftRight{left: x.Base, right: Group(vals...), operator: "IN"})
}

// NOT IN
func (x Common[T, B]) NIN(vals ...any) T {
	return X[T, B](leftRight{left: x.Base, right: Group(vals...), operator: "NOT IN"})
}

// OR
func (x Common[T, B]) OR(target any) T {
	return X[T, B](leftRight{operator: "OR", left: x.Base, right: target})
}

// AND
func (x Common[T, B]) AND(target any) T {
	return X[T, B](leftRight{operator: "AND", left: x.Base, right: target})
}

// Concatenate: `||``
func (x Common[T, B]) CONCAT(target any) T {
	return X[T, B](leftRight{operator: "||", left: x.Base, right: target})
}

// Subtract
func (x Common[T, B]) MINUS(target any) T {
	return X[T, B](leftRight{operator: "-", left: x.Base, right: target})
}

// As does not return a Builder. Should be used at the end of an expression
func (x Common[T, B]) AS(alias string) query.Expression {
	return leftRight{left: x.Base, operator: " AS ", right: Quote(alias)}
}
