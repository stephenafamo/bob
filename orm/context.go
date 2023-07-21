package orm

type ctxKey int

const (
	// The alias of an eager loader's parent
	CtxLoadParentAlias ctxKey = iota
	// A schema to use when non was specified during generation
	CtxUseSchema
)

type (
	// If set to true, query hooks are skipped
	SkipQueryHooksKey struct{}
	// If set to true, model hooks are skipped
	SkipModelHooksKey struct{}
)
