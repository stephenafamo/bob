package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

type Expression = dialect.Expression

//nolint:gochecknoglobals
var bmod = expr.Builder[Expression, Expression]{}

// F creates a function expression with the given name and args
//
//	SQL: generate_series(1, 3)
//	Go: psql.F("generate_series", 1, 3)
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

// SQL: ($1, $2, $3)
// Go: psql.ArgGroup("a", "b", "c")
func ArgGroup(args ...any) Expression {
	return bmod.ArgGroup(args...)
}

// SQL: $1, $2, $3
// Go: psql.Placeholder(3)
func Placeholder(n uint) Expression {
	return bmod.Placeholder(n)
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

// SQL: CAST(a AS int)
// Go: psql.Cast("a", "int")
func Cast(exp bob.Expression, typname string) Expression {
	return bmod.Cast(exp, typname)
}

// SQL: CASE WHEN a THEN b ELSE c END
// Go: psql.Case().When("a", "b").Else("c")
func Case() expr.CaseChain[Expression, Expression] {
	return expr.NewCase[Expression, Expression]()
}

// SQL: EXISTS ((SELECT 1))
// Go: psql.Exists(psql.Select(sm.Columns("1")))
func Exists(exp bob.Expression) Expression {
	return bmod.Exists(exp)
}

// SQL: - 1 - 2
// Go: psql.Minus(psql.Arg(1)).Minus(psql.Arg(2))
func Minus(exp bob.Expression) Expression {
	return bmod.Minus(exp)
}

// SQL: a = ANY((SELECT name FROM users))
// Go: psql.Quote("a").EQ(psql.Any(psql.Select(sm.Columns("name"), sm.From("users"))))
func Any(exp bob.Expression) Expression {
	return bmod.Any(exp)
}

// SQL: a = SOME((SELECT name FROM users))
// Go: psql.Quote("a").EQ(psql.Some(psql.Select(sm.Columns("name"), sm.From("users"))))
func Some(exp bob.Expression) Expression {
	return bmod.Some(exp)
}

// SQL: a = ALL((SELECT name FROM users))
// Go: psql.Quote("a").EQ(psql.All(psql.Select(sm.Columns("name"), sm.From("users"))))
func All(exp bob.Expression) Expression {
	return bmod.All(exp)
}
