package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

type or struct {
	to string
}

func (o *or) SetOr(to string) {
	o.to = to
}

func (o or) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressIf(w, d, start, o.to, o.to != "", " OR ", "")
}

type orMod[Q interface{ SetOr(string) }] struct{}

func (o orMod[Q]) OrAbort() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("ABORT")
	})
}

func (o orMod[Q]) OrFail() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("FAIL")
	})
}

func (o orMod[Q]) OrIgnore() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("IGNORE")
	})
}

func (o orMod[Q]) OrReplace() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("REPLACE")
	})
}

func (o orMod[Q]) OrRollback() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("ROLLBACK")
	})
}

type withMod[Q interface {
	AppendWith(expr.CTE)
	SetRecursive(bool)
}] struct{}

func (withMod[Q]) With(name string, columns ...string) cteChain[Q] {
	return cteChain[Q](func() expr.CTE {
		return expr.CTE{
			Name:    name,
			Columns: columns,
		}
	})
}

func (withMod[Q]) Recursive(r bool) query.Mod[Q] {
	return mods.Recursive[Q](r)
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

func (c cteChain[Q]) NotMaterialized() cteChain[Q] {
	var b = false
	cte := c()
	cte.Materialized = &b
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

func (c cteChain[Q]) Materialized() cteChain[Q] {
	var b = true
	cte := c()
	cte.Materialized = &b
	return cteChain[Q](func() expr.CTE {
		return cte
	})
}

type fromItemMod struct{}

func (fromItemMod) NotIndexed() query.Mod[*expr.FromItem] {
	return mods.QueryModFunc[*expr.FromItem](func(q *expr.FromItem) {
		var s string
		q.IndexedBy = &s
	})
}

func (fromItemMod) IndexedBy(indexName string) query.Mod[*expr.FromItem] {
	return mods.QueryModFunc[*expr.FromItem](func(q *expr.FromItem) {
		q.IndexedBy = &indexName
	})
}

type joinChain[Q interface{ AppendJoin(expr.Join) }] func() expr.Join

func (j joinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (j joinChain[Q]) As(alias string) joinChain[Q] {
	jo := j()
	jo.Alias = alias

	return joinChain[Q](func() expr.Join {
		return jo
	})
}

func (j joinChain[Q]) Natural() query.Mod[Q] {
	jo := j()
	jo.Natural = true

	return mods.Join[Q](jo)
}

func (j joinChain[Q]) On(on ...any) query.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, on)

	return mods.Join[Q](jo)
}

func (j joinChain[Q]) OnEQ(a, b any) query.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, bmod.X(a).EQ(b))

	return mods.Join[Q](jo)
}

func (j joinChain[Q]) Using(using ...any) query.Mod[Q] {
	jo := j()
	jo.Using = using

	return mods.Join[Q](jo)
}

type joinMod[Q interface{ AppendJoin(expr.Join) }] struct{}

func (j joinMod[Q]) InnerJoin(e any) joinChain[Q] {
	return joinChain[Q](func() expr.Join {
		return expr.Join{
			Type: expr.InnerJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) LeftJoin(e any) joinChain[Q] {
	return joinChain[Q](func() expr.Join {
		return expr.Join{
			Type: expr.LeftJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) RightJoin(e any) joinChain[Q] {
	return joinChain[Q](func() expr.Join {
		return expr.Join{
			Type: expr.RightJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) FullJoin(e any) joinChain[Q] {
	return joinChain[Q](func() expr.Join {
		return expr.Join{
			Type: expr.FullJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) CrossJoin(e any) query.Mod[Q] {
	return mods.Join[Q]{
		Type: expr.CrossJoin,
		To:   e,
	}
}

type orderBy[Q interface{ AppendOrder(expr.OrderDef) }] func() expr.OrderDef

func (s orderBy[Q]) Apply(q Q) {
	q.AppendOrder(s())
}

func (o orderBy[Q]) Collate(collation string) orderBy[Q] {
	order := o()
	order.CollationName = collation

	return orderBy[Q](func() expr.OrderDef {
		return order
	})
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

type windowChain[Q interface{ AppendWindow(expr.NamedWindow) }] struct {
	name string
	def  expr.WindowDef
}

func (w *windowChain[Q]) Apply(q Q) {
	q.AppendWindow(expr.NamedWindow{
		Name:      w.name,
		Definiton: w.def,
	})
}

func (w *windowChain[Q]) As(name string) *windowChain[Q] {
	w.def.From = name
	return w
}

func (w *windowChain[Q]) PartitionBy(condition ...any) *windowChain[Q] {
	w.def = w.def.PartitionBy(condition...)
	return w
}

func (w *windowChain[Q]) OrderBy(order ...any) *windowChain[Q] {
	w.def = w.def.OrderBy(order...)
	return w
}
