package psql

import (
	"github.com/stephenafamo/bob/expr"
)

var bmod = builderMod{}

type builderMod struct {
	expr.Builder[chain, chain]
}

func (b builderMod) F(name string, args ...any) *function {
	f := &function{
		name: name,
		args: args,
	}

	// We have embeded the same function as the chain base
	// this is so that chained methods can also be used by functions
	f.Chain.Base = f

	return f
}

type chain struct {
	expr.Chain[chain, chain]
}

func (chain) New(exp any) chain {
	var b chain
	b.Base = exp
	return b
}
