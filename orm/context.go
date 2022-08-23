package orm

type ctxKey string

var (
	CtxLoadPrefix      ctxKey = "load_parent_alias"
	CtxLoadParentAlias ctxKey = "load_prefix"
)
