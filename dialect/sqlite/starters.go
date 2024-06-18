package sqlite

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

type Expression = dialect.Expression

//nolint:gochecknoglobals
var bmod = expr.Builder[Expression, Expression]{}

// F creates a function expression with the given name and args
//
//	SQL: generate_series(1, 3)
//	Go: sqlite.F("generate_series", 1, 3)
func F(name string, args ...any) mods.Moddable[*dialect.Function] {
	f := dialect.NewFunction(name, args...)

	return mods.Moddable[*dialect.Function](func(mods ...bob.Mod[*dialect.Function]) *dialect.Function {
		for _, mod := range mods {
			mod.Apply(f)
		}

		return f
	})
}

// S creates a string literal
// SQL: 'a string'
// Go: sqlite.S("a string")
func S(s string) Expression {
	return bmod.S(s)
}

// SQL: NOT true
// Go: sqlite.Not("true")
func Not(exp bob.Expression) Expression {
	return bmod.Not(exp)
}

// SQL: a OR b OR c
// Go: sqlite.Or("a", "b", "c")
func Or(args ...bob.Expression) Expression {
	return bmod.Or(args...)
}

// SQL: a AND b AND c
// Go: sqlite.And("a", "b", "c")
func And(args ...bob.Expression) Expression {
	return bmod.And(args...)
}

// SQL: a || b || c
// Go: sqlite.Concat("a", "b", "c")
func Concat(args ...bob.Expression) Expression {
	return expr.X[Expression, Expression](expr.Join{Exprs: args, Sep: " || "})
}

// SQL: $1, $2, $3
// Go: sqlite.Args("a", "b", "c")
func Arg(args ...any) Expression {
	return bmod.Arg(args...)
}

// SQL: ($1, $2, $3)
// Go: sqlite.ArgGroup("a", "b", "c")
func ArgGroup(args ...any) Expression {
	return bmod.ArgGroup(args...)
}

// SQL: $1, $2, $3
// Go: sqlite.Placeholder(3)
func Placeholder(n uint) Expression {
	return bmod.Placeholder(n)
}

// SQL: (a, b)
// Go: sqlite.Group("a", "b")
func Group(exps ...bob.Expression) Expression {
	return bmod.Group(exps...)
}

// SQL: "table"."column"
// Go: sqlite.Quote("table", "column")
func Quote(ss ...string) Expression {
	return bmod.Quote(ss...)
}

// SQL: where a = $1
// Go: sqlite.Raw("where a = ?", "something")
func Raw(query string, args ...any) Expression {
	return bmod.Raw(query, args...)
}

// SQL: CAST(a AS int)
// Go: psql.Cast("a", "int")
func Cast(exp bob.Expression, typname string) Expression {
	return bmod.Cast(exp, typname)
}
