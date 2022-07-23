package builder

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

type builder[B any] interface {
	New(any) B
}

// Build an expression
func X[T any, B builder[T]](exp any) T {
	// Wrap in parenthesis if not a raw string or string in quotes
	switch exp.(type) {
	case string, rawString, quoted:
		// expected to be printed as it is
		break
	case args:
		// Often initialized in a context that includes
		// its own parenthesis such as VALUES(...)
		break
	case group, parentheses:
		// Already has its own parentheses
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
type Builder[T any, B builder[T]] struct{}

// Start building an expression
func (e Builder[T, B]) X(exp any) T {
	return X[T, B](exp)
}

// prefix the expression with a NOT
func (e Builder[T, B]) NotX(exp any) T {
	return NotX[T, B](exp)
}

// Or
func (e Builder[T, B]) Or(args ...any) T {
	return e.X(sliceJoin{expr: args, operator: " OR "})
}

// And
func (e Builder[T, B]) And(args ...any) T {
	return e.X(sliceJoin{expr: args, operator: " AND "})
}

// Concatenation `||` operator
func (e Builder[T, B]) CONCAT(ss ...any) T {
	return e.X(sliceJoin{expr: ss, operator: " || "})
}

func (e Builder[T, B]) Placeholder(n uint) T {
	return e.Arg(make([]any, n)...)
}

// OVER: For window functions
func (e Builder[T, B]) OVER(f expr.Function, window any) T {
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
