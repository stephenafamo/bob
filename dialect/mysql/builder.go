package mysql

import (
	"github.com/stephenafamo/bob/expr"
)

type chain struct {
	expr.Chain[chain, chain]
}

func (chain) New(exp any) chain {
	var b chain
	b.Base = exp
	return b
}
