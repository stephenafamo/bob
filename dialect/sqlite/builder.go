package sqlite

import (
	"database/sql"

	"github.com/stephenafamo/bob/expr"
)

var bmod = builderMod{}

type builderMod struct {
	expr.Builder[chain, chain]
}

func (builderMod) F(name string, args ...any) *function {
	f := &function{
		name: name,
		args: args,
	}

	// We have embeded the same function as the chain base
	// this is so that chained methods can also be used by functions
	f.Chain.Base = f

	return f
}

func (builderMod) Named(name string, value any) chain {
	var b chain
	b.Base = sql.Named(name, value)
	return b
}

type chain struct {
	expr.Chain[chain, chain]
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
