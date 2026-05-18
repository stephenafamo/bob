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

// NoSep can be assigned to Join.Sep to join expressions with no separator at all.
// The zero value of Sep (empty string) still defaults to a single space for backward compatibility.
const NoSep = "\x00"

// If Sep is empty, a space is used. Set Sep to NoSep to use no separator.
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
	switch sep {
	case NoSep:
		sep = ""
	case "":
		sep = " "
	}

	return bob.ExpressSliceTo(ctx, w, d, start, s.Exprs, "", sep, "", args)
}

// Glue joins expressions with no separator between them.
//
//	SQL: EXCLUDED."col"
//	Go: expr.Glue(expr.Raw("EXCLUDED."), expr.Quote("col"))
func Glue(exprs ...bob.Expression) Join {
	return Join{Exprs: exprs, Sep: NoSep}
}
