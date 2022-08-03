package mysql

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

type fromItemMod struct {
	mods.TableAliasMod[*clause.FromItem] // Adding an alias to from item
	partitionMod[*clause.FromItem]       // for partitions
}

func (fromItemMod) Lateral() bob.Mod[*clause.FromItem] {
	return mods.QueryModFunc[*clause.FromItem](func(q *clause.FromItem) {
		q.Lateral = true
	})
}

func (fromItemMod) UseIndex(first string, others ...string) *indexHintChain[*clause.FromItem] {
	return &indexHintChain[*clause.FromItem]{
		hint: clause.IndexHint{
			Type:    "USE",
			Indexes: append([]string{first}, others...),
		},
	}
}

func (fromItemMod) IgnoreIndex(first string, others ...string) *indexHintChain[*clause.FromItem] {
	return &indexHintChain[*clause.FromItem]{
		hint: clause.IndexHint{
			Type:    "IGNORE",
			Indexes: append([]string{first}, others...),
		},
	}
}

func (fromItemMod) ForceIndex(first string, others ...string) *indexHintChain[*clause.FromItem] {
	return &indexHintChain[*clause.FromItem]{
		hint: clause.IndexHint{
			Type:    "FORCE",
			Indexes: append([]string{first}, others...),
		},
	}
}

type indexHintChain[Q interface{ AppendIndexHint(clause.IndexHint) }] struct {
	hint clause.IndexHint
}

func (i *indexHintChain[Q]) Apply(q Q) {
	q.AppendIndexHint(i.hint)
}

func (i *indexHintChain[Q]) ForJoin() *indexHintChain[Q] {
	i.hint.For = "JOIN"
	return i
}

func (i *indexHintChain[Q]) ForOrderBy() *indexHintChain[Q] {
	i.hint.For = "ORDER BY"
	return i
}

func (i *indexHintChain[Q]) ForGroupBy() *indexHintChain[Q] {
	i.hint.For = "GROUP BY"
	return i
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

func (j joinMod[Q]) CrossJoin(e any) bob.Mod[Q] {
	return mods.Join[Q]{
		Type: clause.CrossJoin,
		To:   e,
	}
}

func (j joinMod[Q]) StraightJoin(e any) bob.Mod[Q] {
	return mods.Join[Q]{
		Type: clause.StraightJoin,
		To:   e,
	}
}

type orderBy[Q interface{ AppendOrder(clause.OrderDef) }] func() clause.OrderDef

func (s orderBy[Q]) Apply(q Q) {
	q.AppendOrder(s())
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

func (o orderBy[Q]) Collate(collation string) orderBy[Q] {
	order := o()
	order.CollationName = collation

	return orderBy[Q](func() clause.OrderDef {
		return order
	})
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

type lockChain[Q interface{ SetFor(clause.For) }] func() clause.For

func (l lockChain[Q]) Apply(q Q) {
	q.SetFor(l())
}

func (l lockChain[Q]) NoWait() lockChain[Q] {
	lock := l()
	lock.Wait = clause.LockWaitNoWait
	return lockChain[Q](func() clause.For {
		return lock
	})
}

func (l lockChain[Q]) SkipLocked() lockChain[Q] {
	lock := l()
	lock.Wait = clause.LockWaitSkipLocked
	return lockChain[Q](func() clause.For {
		return lock
	})
}

type windowMod[Q interface{ AppendWindow(clause.NamedWindow) }] struct {
	name string
	clause.WindowDef
	windowChain[*windowMod[Q]]
}

func (w windowMod[Q]) Apply(q Q) {
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
