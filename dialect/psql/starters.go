package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
)

type Expression = dialect.Expression

//nolint:gochecknoglobals
var bmod = expr.Builder[Expression, Expression]{}

// F creates a function expression with the given name and args
//
//	SQL: generate_series(1, 3)
//	Go: psql.F("generate_series", 1, 3)
func F(name string, args ...any) *dialect.Function {
	f := dialect.NewFunction(name, args...)

	// We have embedded the same function as the chain base
	// this is so that chained methods can also be used by functions
	f.Chain.Base = &f

	return &f
}

// S creates a string literal
// SQL: 'a string'
// Go: psql.S("a string")
func S(s string) Expression {
	return bmod.S(s)
}

// SQL: NOT true
// Go: psql.Not("true")
func Not(exp bob.Expression) Expression {
	return bmod.Not(exp)
}

// SQL: a OR b OR c
// Go: psql.Or("a", "b", "c")
func Or(args ...bob.Expression) Expression {
	return bmod.Or(args...)
}

// SQL: a AND b AND c
// Go: psql.And("a", "b", "c")
func And(args ...bob.Expression) Expression {
	return bmod.And(args...)
}

// SQL: a || b || c
// Go: psql.Concat("a", "b", "c")
func Concat(args ...bob.Expression) Expression {
	return expr.X[Expression, Expression](expr.Join{Exprs: args, Sep: " || "})
}

// SQL: $1, $2, $3
// Go: psql.Args("a", "b", "c")
func Arg(args ...any) Expression {
	return bmod.Arg(args...)
}

func ArgNamed(names ...string) Expression {
	return bmod.ArgNamed(names...)
}

// SQL: ($1, $2, $3)
// Go: psql.ArgGroup("a", "b", "c")
func ArgGroup(args ...any) Expression {
	return bmod.ArgGroup(args...)
}

func ArgNamedGroup(names ...string) Expression {
	return bmod.ArgNamedGroup(names...)
}

// SQL: $1, $2, $3
// Go: psql.Placeholder(3)
func Placeholder(n uint) Expression {
	return bmod.Placeholder(n)
}

func NamedPlaceholder(names ...string) Expression {
	return bmod.NamedPlaceholder(names...)
}

// SQL: (a, b)
// Go: psql.Group("a", "b")
func Group(exps ...bob.Expression) Expression {
	return bmod.Group(exps...)
}

// SQL: "table"."column"
// Go: psql.Quote("table", "column")
func Quote(ss ...string) Expression {
	return bmod.Quote(ss...)
}

// SQL: where a = $1
// Go: psql.Raw("where a = ?", "something")
func Raw(query string, args ...any) Expression {
	return bmod.Raw(query, args...)
}
