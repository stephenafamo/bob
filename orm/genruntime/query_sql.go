package genruntime

import (
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// BindSQL binds the argument occurrences for a statement. The returned
// function renders any half-open byte range using the same bindings.
func BindSQL(sql string, args ...orm.ArgWithPosition) func(int, int) bob.Expression {
	return func(from, to int) bob.Expression {
		return orm.ArgsToExpression(sql, from, to, slices.Values(args))
	}
}

// SQLGroup binds a tuple's child expressions when the enclosing query is built.
func SQLGroup(sql string, args func() []orm.ArgWithPosition) bob.Expression {
	return orm.ArgsToExpression(sql, 0, len(sql), func(yield func(orm.ArgWithPosition) bool) {
		slices.Values(args())(yield)
	})
}

// SQLGroups collects the tuples of an expanding argument in input order.
func SQLGroups(n int, group func(int) bob.Expression) bob.Expression {
	groups := make([]bob.Expression, n)
	for i := range groups {
		groups[i] = group(i)
	}
	return expr.Join{Exprs: groups, Sep: ", "}
}

// NewModExecQuery connects the original SQL expression and its dialect builder.
func NewModExecQuery[Q, E bob.Expression](e E, d bob.Dialect, typ bob.QueryType,
	build func(...bob.Mod[Q]) bob.BaseQuery[Q], mod bob.ModFunc[Q],
) *orm.ModExecQuery[Q, E] {
	return &orm.ModExecQuery[Q, E]{
		ExecQuery: orm.ExecQuery[E]{BaseQuery: bob.BaseQuery[E]{Expression: e, Dialect: d, QueryType: typ}},
		Build:     build,
		Mod:       mod,
	}
}

// NewModQuery also attaches the mapper used by One, All, Cursor and Each.
func NewModQuery[Tr bob.Transformer[T, Ts], T, Ts any, Q, E bob.Expression](e E, d bob.Dialect, typ bob.QueryType,
	build func(...bob.Mod[Q]) bob.BaseQuery[Q], mod bob.ModFunc[Q], mapper scan.Mapper[T],
) *orm.ModQuery[Q, E, T, Ts, Tr] {
	q := NewModExecQuery(e, d, typ, build, mod)
	return &orm.ModQuery[Q, E, T, Ts, Tr]{
		Query: orm.Query[E, T, Ts, Tr]{ExecQuery: q.ExecQuery, Scanner: mapper},
		Build: build,
		Mod:   mod,
	}
}
