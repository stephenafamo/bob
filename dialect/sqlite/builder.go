package sqlite

import (
	"github.com/stephenafamo/bob/builder"
)

var bmod = builderMod{}

type builderMod struct {
	builder.Builder[chain, chain]
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
	builder.Chain[chain, chain]
}

func (c chain) Get() any {
	return c.Base
}

func (chain) New(exp any) chain {
	var b chain
	b.Base = exp
	return b
}

func (f function) NewFunction(name string, args ...any) function {
	return function{
		name: name,
		args: args,
	}
}
