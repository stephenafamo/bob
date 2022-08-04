package expr

type builder[B any] interface {
	New(any) B
}

// Build an expression
func X[T any, B builder[T]](exp any, others ...any) T {
	var b B

	// Easy chain. For example:
	// X("a", "=", "b")
	if len(others) > 0 {
		exp = Join{Exprs: append([]any{exp}, others...)}
	}

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

	return b.New(exp)
}

// prefix the expression with a NOT
func Not[T any, B builder[T]](exp any) T {
	var b B
	return b.New(P(Join{Exprs: []any{"NOT", X[T, B](exp)}}))
}

// To be embedded in query mods
// T is the chain type, this allows dialects to have custom chain methods
// F is function type, so that the dialect can change where it
// accepted. E.g. it can be modified to work as a mod
// B has a New() method that is used to create a new instance of T
type Builder[T any, B builder[T]] struct{}

// Start building an expression
func (e Builder[T, B]) X(exp any, others ...any) T {
	return X[T, B](exp, others...)
}

// prefix the expression with a NOT
func (e Builder[T, B]) Not(exp any) T {
	return Not[T, B](exp)
}

// Or
func (e Builder[T, B]) Or(args ...any) T {
	return e.X(Join{Exprs: args, Sep: " OR "})
}

// And
func (e Builder[T, B]) And(args ...any) T {
	return e.X(Join{Exprs: args, Sep: " AND "})
}

// Concatenation `||` operator
func (e Builder[T, B]) Concat(ss ...any) T {
	return e.X(Join{Exprs: ss, Sep: " || "})
}

// single quoted raw string
func (e Builder[T, B]) S(s string) T {
	return e.X(rawString(s))
}

// Comma separated list of arguments
func (e Builder[T, B]) Arg(vals ...any) T {
	return e.X(args{vals: vals})
}

func (e Builder[T, B]) Placeholder(n uint) T {
	return e.Arg(make([]any, n)...)
}

func (e Builder[T, B]) Raw(query string, args ...any) T {
	return e.X(Raw{
		query: query,
		args:  args,
	})
}

func (e Builder[T, B]) Group(exps ...any) T {
	return e.X(group(exps))
}

// quoted and joined... something like "users"."id"
func (e Builder[T, B]) Quote(aa ...string) T {
	ss := make([]any, len(aa))
	for k, v := range aa {
		ss[k] = v
	}

	return e.X(quoted(ss))
}

// Add parentheses around an expression
func (e Builder[T, B]) P(exp any) T {
	return e.X(parentheses{inside: exp})
}
