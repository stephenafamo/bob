package builder

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
func OP(operator string, left, right any) query.Expression {
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

type StartEnd struct {
	prefix string
	expr   any
	suffix string
}

func (i StartEnd) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.ExpressIf(w, d, start, i.expr, true, i.prefix, i.suffix)
	if err != nil {
		return nil, err
	}

	return args, nil
}
