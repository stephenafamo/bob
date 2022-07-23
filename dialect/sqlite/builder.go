package sqlite

import "github.com/stephenafamo/bob/builder"

var bmod = BuilderMod{}

type BuilderMod = builder.Builder[Builder, Builder]

type Builder struct {
	builder.Chain[Builder, Builder]
}

func (Builder) New(exp any) Builder {
	var b Builder
	b.Base = exp
	return b
}
