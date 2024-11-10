package bob

var (
	_ Expression = &cached{}

	_ Query = BaseQuery[Expression]{}
	_ Query = BoundQuery[Expression]{}

	_ Loadable = BaseQuery[Expression]{}
	_ Loadable = BoundQuery[Expression]{}
	_ Loadable = &cached{}

	_ MapperModder = BaseQuery[Expression]{}
	_ MapperModder = BoundQuery[Expression]{}
	_ MapperModder = &cached{}

	_ HookableQuery = BaseQuery[Expression]{}
	_ HookableQuery = BoundQuery[Expression]{}
)
