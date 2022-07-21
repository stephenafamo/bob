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
	largs, err := query.Express(w, d, start, lr.left)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(w, " %s ", lr.operator)

	rargs, err := query.Express(w, d, start+len(largs), lr.right)
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

type sliceJoin struct {
	expr     []any
	operator string
}

func (s sliceJoin) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, s.expr, "", s.operator, "")
}

// OR
func OR(args ...any) query.Expression {
	return sliceJoin{
		expr:     args,
		operator: " OR ",
	}
}

// AND
func AND(args ...any) query.Expression {
	return sliceJoin{
		expr:     args,
		operator: " AND ",
	}
}

// Concatenation `||` operator
func CONCAT(ss ...any) query.Expression {
	return sliceJoin{
		expr:     ss,
		operator: " || ",
	}
}

type startEnd struct {
	prefix string
	expr   any
	suffix string
}

func (i startEnd) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.ExpressIf(w, d, start, i.expr, true, i.prefix, i.suffix)
	if err != nil {
		return nil, err
	}

	return args, nil
}

// OVER: For window functions
func OVER(f Function, window any) query.Expression {
	return query.ExpressionFunc(func(w io.Writer, d query.Dialect, start int) ([]any, error) {
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
	})
}
