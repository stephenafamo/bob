package expr

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/query"
)

// An operator that has a left and right side
type leftRight struct {
	operator string
	right    any
	left     any
}

func (lr leftRight) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	largs, err := query.Express(w, d, start, lr.right)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(w, " %s ", lr.operator)

	rargs, err := query.Express(w, d, start+len(largs), lr.left)
	if err != nil {
		return nil, err
	}

	return append(largs, rargs...), nil
}

// Generic operator between a left and right val
func OP(operator string, right, left any) query.Expression {
	return leftRight{
		right:    right,
		left:     left,
		operator: operator,
	}
}

// Equality operator
func EQ(right, left any) query.Expression {
	return leftRight{
		right:    right,
		left:     left,
		operator: "=",
	}
}

// Inequality operator
func NE(right, left any) query.Expression {
	return leftRight{
		right:    right,
		left:     left,
		operator: "<>",
	}
}

// Subtract
func MINUS(right, left any) query.Expression {
	return leftRight{
		right:    right,
		left:     left,
		operator: "-",
	}
}

// IN operator
// if the first value is in any of the rest
func IN(right any, left ...any) query.Expression {
	return leftRight{
		right:    right,
		left:     Group(left...),
		operator: "IN",
	}
}

// NOT IN operator
// if the first value is not in any of the rest
func NIN(right any, left ...any) query.Expression {
	return leftRight{
		right:    right,
		left:     Group(left...),
		operator: "NOT IN",
	}
}

func NULL(exp any) query.Expression {
	return null{
		expr:   exp,
		isNull: true,
	}
}

func NOTNULL(exp any) query.Expression {
	return null{
		expr:   exp,
		isNull: false,
	}
}

type null struct {
	expr   any
	isNull bool
}

func (n null) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.Express(w, d, start, n.expr)
	if err != nil {
		return nil, err
	}

	w.Write([]byte(" IS"))
	if !n.isNull {
		w.Write([]byte(" NOT"))
	}
	w.Write([]byte(" NULL"))

	return args, nil
}

// For window functions
func OVER(f Function, window any) query.Expression {
	return over{
		function: f,
		window:   window,
	}
}

type over struct {
	function Function
	window   any
}

func (o over) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	largs, err := query.Express(w, d, start, o.function)
	if err != nil {
		return nil, err
	}

	fmt.Fprint(w, " OVER (")

	rargs, err := query.Express(w, d, start+len(largs), o.window)
	if err != nil {
		return nil, err
	}

	fmt.Fprint(w, ")")

	return append(largs, rargs...), nil
}

// Concatenation `||` operator
func CONCAT(ss ...any) query.Expression {
	return concat(ss)
}

type concat []any

func (c concat) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, c, "", " || ", "")
}
