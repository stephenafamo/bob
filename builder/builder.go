package builder

type builder[B any] interface {
	New(any) B
}

type functionBuilder[F any] interface {
	NewFunction(name string, args ...any) F
}

// Build an expression
func X[T any, B builder[T]](exp any) T {
	var b B

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
func NotX[T any, B builder[T]](exp any) T {
	var b B
	return b.New(P(StartEnd{prefix: "NOT ", expr: X[T, B](exp)}))
}

// To be embeded in query mods
// T is the chain type, this allows dialects to have custom chain methods
// F is function type, so that the dialect can change where it
// accepted. E.g. it can be modified to work as a mod
// B has a New() method that is used to create a new instance of T
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
