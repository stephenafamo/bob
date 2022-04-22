package psql

import (
	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/mods"
	"github.com/stephenafamo/typesql/query"
)

type join[Q interface{ AppendJoin(expr.Join) }] func() expr.Join

func (j join[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (j join[Q]) To(e any) join[Q] {
	jo := j()
	jo.To = e

	return join[Q](func() expr.Join {
		return jo
	})
}

func (j join[Q]) As(alias string) join[Q] {
	jo := j()
	jo.Alias = alias

	return join[Q](func() expr.Join {
		return jo
	})
}

func (j join[Q]) Natural() mods.QueryMod[Q] {
	jo := j()
	jo.Natural = true

	return mods.Join[Q](jo)
}

func (j join[Q]) On(on ...any) mods.QueryMod[Q] {
	jo := j()
	jo.On = append(jo.On, on)

	return mods.Join[Q](jo)
}

func (j join[Q]) OnEQ(a, b any) mods.QueryMod[Q] {
	jo := j()
	jo.On = append(jo.On, expr.EQ(a, b))

	return mods.Join[Q](jo)
}

func (j join[Q]) Using(using ...any) mods.QueryMod[Q] {
	jo := j()
	jo.Using = using

	return mods.Join[Q](jo)
}

type joinMod[Q interface{ AppendJoin(expr.Join) }] struct{}

func (j joinMod[Q]) InnerJoin(e any) join[Q] {
	return join[Q](func() expr.Join {
		return expr.Join{
			Type: expr.InnerJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) LeftJoin(e any) join[Q] {
	return join[Q](func() expr.Join {
		return expr.Join{
			Type: expr.LeftJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) RightJoin(e any) join[Q] {
	return join[Q](func() expr.Join {
		return expr.Join{
			Type: expr.RightJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) FullJoin(e any) join[Q] {
	return join[Q](func() expr.Join {
		return expr.Join{
			Type: expr.FullJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) CrossJoin(e any) mods.QueryMod[Q] {
	return mods.Join[Q]{
		Type: expr.CrossJoin,
		To:   e,
	}
}

type orderBy[Q interface{ AppendOrder(expr.OrderDef) }] func() expr.OrderDef

func (s orderBy[Q]) Apply(q Q) {
	q.AppendOrder(s())
}

func (o orderBy[Q]) Asc() orderBy[Q] {
	order := o()
	order.Direction = "ASC"

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
}

func (o orderBy[Q]) Desc() orderBy[Q] {
	order := o()
	order.Direction = "DESC"

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
}

func (o orderBy[Q]) Using(operator string) orderBy[Q] {
	order := o()
	order.Direction = "USING " + operator

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
}

func (o orderBy[Q]) NullsFirst() orderBy[Q] {
	order := o()
	order.Nulls = "FIRST"

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
}

func (o orderBy[Q]) NullsLast() orderBy[Q] {
	order := o()
	order.Nulls = "LAST"

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
}

func (o orderBy[Q]) Collate(collation string) orderBy[Q] {
	order := o()
	order.CollationName = collation

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
}

type onConflict[Q interface{ SetConflict(expr.Conflict) }] func() expr.Conflict

func (s onConflict[Q]) Apply(q Q) {
	q.SetConflict(s())
}

func (c onConflict[Q]) On(target any, where ...any) onConflict[Q] {
	conflict := c()
	conflict.Target.Target = target
	conflict.Target.Where = append(conflict.Target.Where, where...)

	return onConflict[Q](func() expr.Conflict {
		return conflict
	})
}

func (c onConflict[Q]) DoNothing() mods.QueryMod[Q] {
	conflict := c()
	conflict.Do = "NOTHING"

	return onConflict[Q](func() expr.Conflict {
		return conflict
	})
}

func (c onConflict[Q]) DoUpdate() onConflict[Q] {
	conflict := c()
	conflict.Do = "UPDATE"

	return onConflict[Q](func() expr.Conflict {
		return conflict
	})
}

func (c onConflict[Q]) Set(set any) onConflict[Q] {
	conflict := c()
	conflict.Set.Set = append(conflict.Set.Set, set)

	return onConflict[Q](func() expr.Conflict {
		return conflict
	})
}

func (c onConflict[Q]) SetEQ(a, b any) onConflict[Q] {
	conflict := c()
	conflict.Set.Set = append(conflict.Set.Set, expr.EQ(a, b))

	return onConflict[Q](func() expr.Conflict {
		return conflict
	})
}

func (c onConflict[Q]) Where(where ...any) onConflict[Q] {
	conflict := c()
	conflict.Where.Conditions = append(conflict.Where.Conditions, where...)

	return onConflict[Q](func() expr.Conflict {
		return conflict
	})
}

type cteChain[Q interface{ AppendWith(expr.CTE) }] func() expr.CTE

func (c cteChain[Q]) Apply(q Q) {
	q.AppendWith(c())
}

func (c cteChain[Q]) Name(tableName string, columnNames ...string) cteChain[Q] {
	cte := c()
	cte.Name = tableName
	cte.Columns = columnNames
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) As(q query.Query) cteChain[Q] {
	cte := c()
	cte.Query = q
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) Materialized(b bool) cteChain[Q] {
	cte := c()
	cte.Materialized = &b
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) SearchBreadth(setCol string, searchCols ...string) cteChain[Q] {
	cte := c()
	cte.Search = expr.CTESearch{
		Order:   expr.SearchDepth,
		Columns: searchCols,
		Set:     setCol,
	}
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) SearchDepth(setCol string, searchCols ...string) cteChain[Q] {
	cte := c()
	cte.Search = expr.CTESearch{
		Order:   expr.SearchDepth,
		Columns: searchCols,
		Set:     setCol,
	}
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) Cycle(set, using string, cols ...string) cteChain[Q] {
	cte := c()
	cte.Cycle.Set = set
	cte.Cycle.Using = using
	cte.Cycle.Columns = cols
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) CycleSet(value, defaultVal any) cteChain[Q] {
	cte := c()
	cte.Cycle.SetVal = value
	cte.Cycle.DefaultVal = defaultVal
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}
