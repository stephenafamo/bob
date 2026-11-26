package bob

var (
	_ Expression = &cached{}

	_ Query = QueryBuilder[int]{}
	_ Query = ExpressionQuery[Expression]{}
	_ Query = BoundQuery[Expression]{}

	_ Loadable = ExpressionQuery[Expression]{}
	_ Loadable = BoundQuery[Expression]{}
	_ Loadable = &cached{}

	_ MapperModder = ExpressionQuery[Expression]{}
	_ MapperModder = BoundQuery[Expression]{}
	_ MapperModder = &cached{}

	_ HookableQuery = ExpressionQuery[Expression]{}
	_ HookableQuery = BoundQuery[Expression]{}
)
