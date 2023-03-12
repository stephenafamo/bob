package expr

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
)

// An operator that has a left and right side
type leftRight struct {
	operator string
	right    any
	left     any
}

func (lr leftRight) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	largs, err := bob.Express(w, d, start, lr.left)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(w, " %s ", lr.operator)

	rargs, err := bob.Express(w, d, start+len(largs), lr.right)
	if err != nil {
		return nil, err
	}

	return append(largs, rargs...), nil
}

// Generic operator between a left and right val
func OP(operator string, left, right any) bob.Expression {
	return leftRight{
		right:    right,
		left:     left,
		operator: operator,
	}
}

// If no separator, a space is used
type Join struct {
	Exprs []bob.Expression
	Sep   string
}

func (s Join) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	sep := s.Sep
	if sep == "" {
		sep = " "
	}

	return bob.ExpressSlice(w, d, start, s.Exprs, "", sep, "")
}
