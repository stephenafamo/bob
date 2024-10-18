package orm

type ctxKey int

const (
	// A schema to use when non was specified during generation
	CtxUseSchema ctxKey = iota
)
