package genruntime

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

// ValueOrDefault reads a setter at build time, after before-insert hooks.
// An omitted value renders DEFAULT; an explicit NULL is a bound argument.
func ValueOrDefault(f func() (any, bool)) bob.Expression {
	return bob.ExpressionFunc(func(_ context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		if v, set := f(); set {
			d.WriteArg(w, start)
			return []any{v}, nil
		}
		_, err := w.WriteString("DEFAULT")
		return nil, err
	})
}

// ValueOrNull is the SQLite variant, where omitted values in a VALUES
// tuple must bind NULL instead of using a DEFAULT expression.
func ValueOrNull(f func() (any, bool)) bob.Expression {
	return bob.ExpressionFunc(func(_ context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		v, set := f()
		if !set {
			v = nil
		}
		d.WriteArg(w, start)
		return []any{v}, nil
	})
}

// SetField builds a qualified assignment for a supplied setter field.
func SetField(prefix []string, column string, value any) bob.Expression {
	return expr.OP("=", expr.Quote(append(prefix, column)...), expr.Arg(value))
}
