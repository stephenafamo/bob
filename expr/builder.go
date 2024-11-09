package expr

import (
	"github.com/stephenafamo/bob"
)

type builder[B any] interface {
	New(bob.Expression) B
}

// Build an expression
func X[T bob.Expression, B builder[T]](exp bob.Expression, others ...bob.Expression) T {
	var b B

	// Easy chain. For example:
	// X("a", "=", "b")
	if len(others) > 0 {
		exp = Join{Exprs: append([]bob.Expression{exp}, others...)}
	}

	// Wrap in parenthesis if not a raw string or string in quotes
	switch t := exp.(type) {
	case Clause, Raw, rawString, quoted:
		// expected to be printed as it is
		break
	case args:
		// Often initialized in a context that includes
		// its own parenthesis such as VALUES(...)
		break
	case group:
		// Already has its own parentheses
		break
	case T:
		return t
	default:
		exp = group{exp}
	}

	return b.New(exp)
}

// prefix the expression with a NOT
func Not[T bob.Expression, B builder[T]](exp bob.Expression) T {
	var b B
	return b.New(Join{Exprs: []bob.Expression{not, X[T, B](exp)}})
}

// To be embedded in query mods
// T is the chain type, this allows dialects to have custom chain methods
// F is function type, so that the dialect can change where it
// accepted. E.g. it can be modified to work as a mod
// B has a New() method that is used to create a new instance of T
type Builder[T bob.Expression, B builder[T]] struct{}

// prefix the expression with a NOT
func (e Builder[T, B]) Not(exp bob.Expression) T {
	return Not[T, B](exp)
}

// Or
func (e Builder[T, B]) Or(args ...bob.Expression) T {
	return X[T, B](Join{Exprs: args, Sep: " OR "})
}

// And
func (e Builder[T, B]) And(args ...bob.Expression) T {
	return X[T, B](Join{Exprs: args, Sep: " AND "})
}

// single quoted raw string
func (e Builder[T, B]) S(s string) T {
	return X[T, B](rawString(s))
}

// Comma separated list of arguments
func (e Builder[T, B]) Arg(vals ...any) T {
	return X[T, B](Arg(vals...))
}

// Comma separated list of arguments surrounded by parentheses
func (e Builder[T, B]) ArgGroup(vals ...any) T {
	return X[T, B](ArgGroup(vals...))
}

func (e Builder[T, B]) Placeholder(n uint) T {
	return e.Arg(make([]any, n)...)
}

func (e Builder[T, B]) Raw(query string, args ...any) T {
	return X[T, B](Clause{
		query: query,
		args:  args,
	})
}

// Add parentheses around an expressions and separate them by commas
func (e Builder[T, B]) Group(exps ...bob.Expression) T {
	return X[T, B](group(exps))
}

// quoted and joined... something like "users"."id"
func (e Builder[T, B]) Quote(aa ...string) T {
	return X[T, B](Quote(aa...))
}

// quoted and joined... something like "users"."id"
func (e Builder[T, B]) Cast(exp bob.Expression, typname string) T {
	return X[T, B](Cast(exp, typname))
}
