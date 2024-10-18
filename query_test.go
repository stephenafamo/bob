package bob

var (
	_ Loadable     = BaseQuery[Expression]{}
	_ MapperModder = BaseQuery[Expression]{}
)
