package orm

type ctxKey string

//nolint:gochecknoglobals
var (
	CtxLoadPrefix      ctxKey = "load_parent_alias"
	CtxLoadParentAlias ctxKey = "load_prefix"
)
