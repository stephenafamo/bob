package sqlite

import "github.com/stephenafamo/bob/expr"

var bmod = BuilderMod{}

type BuilderMod = expr.ExpressionBuilder[Builder, Builder]

type Builder struct {
	expr.Common[Builder, Builder]
}

func (Builder) New(exp any) Builder {
	var b Builder
	b.Base = exp
	return b
}
