package bob

var (
	_ Query         = BaseQuery[Expression]{}
	_ Loadable      = BaseQuery[Expression]{}
	_ MapperModder  = BaseQuery[Expression]{}
	_ HookableQuery = BaseQuery[Expression]{}
)
