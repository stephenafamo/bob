package psql

import (
	"github.com/stephenafamo/bob/expr"
)

var bmod = expr.Builder[chain, chain]{}

type chain struct {
	expr.Chain[chain, chain]
}

func (chain) New(exp any) chain {
	var b chain
	b.Base = exp
	return b
}

// BETWEEN SYMMETRIC a AND b
func (x chain) BetweenSymmetric(a, b any) chain {
	return bmod.X(expr.Join{Exprs: []any{
		x.Base, "BETWEEN SYMMETRIC", a, "AND", b,
	}})
}

// NOT BETWEEN SYMMETRIC a AND b
func (x chain) NotBetweenSymmetric(a, b any) chain {
	return bmod.X(expr.Join{Exprs: []any{
		x.Base, "NOT BETWEEN SYMMETRIC", a, "AND", b,
	}})
}
