package orm

type ctxKey int

const (
	// A schema to use when non was specified during generation
	CtxUseSchema ctxKey = iota
)

type (
	// If set to true, query hooks are skipped
	SkipQueryHooksKey struct{}
	// If set to true, model hooks are skipped
	SkipModelHooksKey struct{}
)
