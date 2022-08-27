package orm

type ctxKey int

const (
	// The prefix of an eager loaded relationship
	CtxLoadPrefix ctxKey = iota
	// The alias of an eager loader's parent
	CtxLoadParentAlias
	// If set to true, hooks are skipped
	ctxSkipHooks
)
