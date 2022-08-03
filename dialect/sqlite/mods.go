package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

type withMod[Q interface {
	AppendWith(clause.CTE)
	SetRecursive(bool)
}] struct{}

func (withMod[Q]) With(name string, columns ...string) cteChain[Q] {
	return cteChain[Q](func() clause.CTE {
		return clause.CTE{
			Name:    name,
			Columns: columns,
		}
	})
}

func (withMod[Q]) Recursive(r bool) bob.Mod[Q] {
	return mods.Recursive[Q](r)
}

type cteChain[Q interface{ AppendWith(clause.CTE) }] func() clause.CTE

func (c cteChain[Q]) Apply(q Q) {
	q.AppendWith(c())
}

func (c cteChain[Q]) As(q bob.Query) cteChain[Q] {
	cte := c()
	cte.Query = q
	return cteChain[Q](func() clause.CTE {
		return cte
	})
}

func (c cteChain[Q]) NotMaterialized() cteChain[Q] {
	var b = false
	cte := c()
	cte.Materialized = &b
	return cteChain[Q](func() clause.CTE {
		return cte
	})
}

func (c cteChain[Q]) Materialized() cteChain[Q] {
	var b = true
	cte := c()
	cte.Materialized = &b
	return cteChain[Q](func() clause.CTE {
		return cte
	})
}

type fromItemMod struct{}

func (fromItemMod) NotIndexed() bob.Mod[*clause.FromItem] {
	return mods.QueryModFunc[*clause.FromItem](func(q *clause.FromItem) {
		var s string
		q.IndexedBy = &s
	})
}

func (fromItemMod) IndexedBy(indexName string) bob.Mod[*clause.FromItem] {
	return mods.QueryModFunc[*clause.FromItem](func(q *clause.FromItem) {
		q.IndexedBy = &indexName
	})
}

type joinChain[Q interface{ AppendJoin(clause.Join) }] func() clause.Join

func (j joinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (j joinChain[Q]) As(alias string) joinChain[Q] {
	jo := j()
	jo.Alias = alias

	return joinChain[Q](func() clause.Join {
		return jo
	})
}

func (j joinChain[Q]) Natural() bob.Mod[Q] {
	jo := j()
	jo.Natural = true

	return mods.Join[Q](jo)
}

func (j joinChain[Q]) On(on ...any) bob.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, on)

	return mods.Join[Q](jo)
}

func (j joinChain[Q]) OnEQ(a, b any) bob.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, bmod.X(a).EQ(b))

	return mods.Join[Q](jo)
}

func (j joinChain[Q]) Using(using ...any) bob.Mod[Q] {
	jo := j()
	jo.Using = using

	return mods.Join[Q](jo)
}

type joinMod[Q interface{ AppendJoin(clause.Join) }] struct{}

func (j joinMod[Q]) InnerJoin(e any) joinChain[Q] {
	return joinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.InnerJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) LeftJoin(e any) joinChain[Q] {
	return joinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.LeftJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) RightJoin(e any) joinChain[Q] {
	return joinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.RightJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) FullJoin(e any) joinChain[Q] {
	return joinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.FullJoin,
			To:   e,
		}
	})
}

func (j joinMod[Q]) CrossJoin(e any) bob.Mod[Q] {
	return mods.Join[Q]{
		Type: clause.CrossJoin,
		To:   e,
	}
}

type orderBy[Q interface{ AppendOrder(clause.OrderDef) }] func() clause.OrderDef

func (s orderBy[Q]) Apply(q Q) {
	q.AppendOrder(s())
}

func (o orderBy[Q]) Collate(collation string) orderBy[Q] {
	order := o()
	order.CollationName = collation

	return orderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o orderBy[Q]) Asc() orderBy[Q] {
	order := o()
	order.Direction = "ASC"

	return orderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o orderBy[Q]) Desc() orderBy[Q] {
	order := o()
	order.Direction = "DESC"

	return orderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o orderBy[Q]) NullsFirst() orderBy[Q] {
	order := o()
	order.Nulls = "FIRST"

	return orderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o orderBy[Q]) NullsLast() orderBy[Q] {
	order := o()
	order.Nulls = "LAST"

	return orderBy[Q](func() clause.OrderDef {
		return order
	})
}

type windowMod[Q interface{ AppendWindow(clause.NamedWindow) }] struct {
	name string
	clause.WindowDef
	windowChain[*windowMod[Q]]
}

func (w *windowMod[Q]) Apply(q Q) {
	q.AppendWindow(clause.NamedWindow{
		Name:       w.name,
		Definition: w.WindowDef,
	})
}

type windowChain[T clause.IWindow] struct {
	def T
}

func (w *windowChain[T]) From(name string) T {
	w.def.SetFrom(name)
	return w.def
}

func (w *windowChain[T]) PartitionBy(condition ...any) T {
	w.def.AddPartitionBy(condition...)
	return w.def
}

func (w *windowChain[T]) OrderBy(order ...any) T {
	w.def.AddOrderBy(order...)
	return w.def
}

func (w *windowChain[T]) Range() T {
	w.def.SetMode("RANGE")
	return w.def
}

func (w *windowChain[T]) Rows() T {
	w.def.SetMode("ROWS")
	return w.def
}

func (w *windowChain[T]) Groups() T {
	w.def.SetMode("GROUPS")
	return w.def
}

func (w *windowChain[T]) FromUnboundedPreceding() T {
	w.def.SetStart("UNBOUNDED PRECEDING")
	return w.def
}

func (w *windowChain[T]) FromPreceding(exp any) T {
	w.def.SetStart(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) (args []any, err error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " PRECEDING")
		}),
	)
	return w.def
}

func (w *windowChain[T]) FromCurrentRow() T {
	w.def.SetStart("CURRENT ROW")
	return w.def
}

func (w *windowChain[T]) FromFollowing(exp any) T {
	w.def.SetStart(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) (args []any, err error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " FOLLOWING")
		}),
	)
	return w.def
}

func (w *windowChain[T]) ToPreceding(exp any) T {
	w.def.SetEnd(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) (args []any, err error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " PRECEDING")
		}),
	)
	return w.def
}

func (w *windowChain[T]) ToCurrentRow(count int) T {
	w.def.SetEnd("CURRENT ROW")
	return w.def
}

func (w *windowChain[T]) ToFollowing(exp any) T {
	w.def.SetEnd(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) (args []any, err error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " FOLLOWING")
		}),
	)
	return w.def
}

func (w *windowChain[T]) ToUnboundedFollowing() T {
	w.def.SetEnd("UNBOUNDED FOLLOWING")
	return w.def
}

func (w *windowChain[T]) ExcludeNoOthers() T {
	w.def.SetExclusion("NO OTHERS")
	return w.def
}

func (w *windowChain[T]) ExcludeCurrentRow() T {
	w.def.SetExclusion("CURRENT ROW")
	return w.def
}

func (w *windowChain[T]) ExcludeGroup() T {
	w.def.SetExclusion("GROUP")
	return w.def
}

func (w *windowChain[T]) ExcludeTies() T {
	w.def.SetExclusion("TIES")
	return w.def
}
