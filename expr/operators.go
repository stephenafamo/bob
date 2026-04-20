package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// An operator that has a left and right side
type leftRight struct {
	operator string
	right    any
	left     any
}

func (lr leftRight) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	err := lr.WriteSQLTo(ctx, w, d, start, &args)
	return args, err
}

func (lr leftRight) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	baseLen := len(*args)
	if err := bob.ExpressTo(ctx, w, d, start, lr.left, args); err != nil {
		return err
	}

	w.WriteString(" ")
	w.WriteString(lr.operator)
	w.WriteString(" ")

	return bob.ExpressTo(ctx, w, d, start+len(*args)-baseLen, lr.right, args)
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

func (s Join) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	err := s.WriteSQLTo(ctx, w, d, start, &args)
	return args, err
}

func (s Join) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	sep := s.Sep
	if sep == "" {
		sep = " "
	}

	return bob.ExpressSliceTo(ctx, w, d, start, s.Exprs, "", sep, "", args)
}
