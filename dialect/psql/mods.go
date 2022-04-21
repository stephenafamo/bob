package psql

import (
	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/query"
)

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

func (c onConflict[Q]) Do(do string) onConflict[Q] {
	conflict := c()
	conflict.Do = do

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
