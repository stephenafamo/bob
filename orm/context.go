package orm

type ctxKey int

const (
	// The alias of an eager loader's parent
	CtxLoadParentAlias ctxKey = iota
	// A schema to use when non was specified during generation
	CtxUseSchema
	// If set to true, hooks are skipped
	ctxSkipHooks
)
