package psql

import "github.com/stephenafamo/bob/builder"

type BuilderMod = builder.Builder[Builder, Builder]

type Builder struct {
	builder.Chain[Builder, Builder]
}

func (Builder) New(exp any) Builder {
	var b Builder
	b.Base = exp
	return b
}
