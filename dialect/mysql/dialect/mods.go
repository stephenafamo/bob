package dialect

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

//nolint:gochecknoglobals
var bmod = expr.Builder[Expression, Expression]{}

func With[Q interface{ AppendWith(clause.CTE) }](name string, columns ...string) CTEChain[Q] {
	return CTEChain[Q](func() clause.CTE {
		return clause.CTE{
			Name:    name,
			Columns: columns,
		}
	})
}

func As[Q interface{ SetTableAlias(string, ...string) }](alias string, columns ...string) bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(q Q) {
		q.SetTableAlias(alias, columns...)
	})
}

func Lateral[Q interface{ SetLateral(bool) }]() bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(q Q) {
		q.SetLateral(true)
	})
}

func UseIndex[Q interface{ AppendIndexHint(clause.IndexHint) }](first string, others ...string) *IndexHintChain[Q] {
	return &IndexHintChain[Q]{
		hint: clause.IndexHint{
			Type:    "USE",
			Indexes: append([]string{first}, others...),
		},
	}
}

func IgnoreIndex[Q interface{ AppendIndexHint(clause.IndexHint) }](first string, others ...string) *IndexHintChain[Q] {
	return &IndexHintChain[Q]{
		hint: clause.IndexHint{
			Type:    "IGNORE",
			Indexes: append([]string{first}, others...),
		},
	}
}

func ForceIndex[Q interface{ AppendIndexHint(clause.IndexHint) }](first string, others ...string) *IndexHintChain[Q] {
	return &IndexHintChain[Q]{
		hint: clause.IndexHint{
			Type:    "FORCE",
			Indexes: append([]string{first}, others...),
		},
	}
}

func Partition[Q interface{ AppendPartition(...string) }](partitions ...string) bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendPartition(partitions...)
	})
}

type IndexHintChain[Q interface{ AppendIndexHint(clause.IndexHint) }] struct {
	hint clause.IndexHint
}

func (i *IndexHintChain[Q]) Apply(q Q) {
	q.AppendIndexHint(i.hint)
}

func (i *IndexHintChain[Q]) ForJoin() *IndexHintChain[Q] {
	i.hint.For = "JOIN"
	return i
}

func (i *IndexHintChain[Q]) ForOrderBy() *IndexHintChain[Q] {
	i.hint.For = "ORDER BY"
	return i
}

func (i *IndexHintChain[Q]) ForGroupBy() *IndexHintChain[Q] {
	i.hint.For = "GROUP BY"
	return i
}

type JoinChain[Q interface{ AppendJoin(clause.Join) }] func() clause.Join

func (j JoinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (j JoinChain[Q]) As(alias string) JoinChain[Q] {
	jo := j()
	jo.Alias = alias

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (j JoinChain[Q]) Natural() bob.Mod[Q] {
	jo := j()
	jo.Natural = true

	return mods.Join[Q](jo)
}

func (j JoinChain[Q]) On(on ...any) bob.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, on)

	return mods.Join[Q](jo)
}

func (j JoinChain[Q]) OnEQ(a, b any) bob.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, bmod.X(a).EQ(b))

	return mods.Join[Q](jo)
}

func (j JoinChain[Q]) Using(using ...any) bob.Mod[Q] {
	jo := j()
	jo.Using = using

	return mods.Join[Q](jo)
}

type joinable interface{ AppendJoin(clause.Join) }

func InnerJoin[Q joinable](e any) JoinChain[Q] {
	return JoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.InnerJoin,
			To:   e,
		}
	})
}

func LeftJoin[Q joinable](e any) JoinChain[Q] {
	return JoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.LeftJoin,
			To:   e,
		}
	})
}

func RightJoin[Q joinable](e any) JoinChain[Q] {
	return JoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.RightJoin,
			To:   e,
		}
	})
}

func CrossJoin[Q joinable](e any) bob.Mod[Q] {
	return mods.Join[Q]{
		Type: clause.CrossJoin,
		To:   e,
	}
}

func StraightJoin[Q joinable](e any) bob.Mod[Q] {
	return mods.Join[Q]{
		Type: clause.StraightJoin,
		To:   e,
	}
}

type OrderBy[Q interface{ AppendOrder(clause.OrderDef) }] func() clause.OrderDef

func (s OrderBy[Q]) Apply(q Q) {
	q.AppendOrder(s())
}

func (o OrderBy[Q]) Asc() OrderBy[Q] {
	order := o()
	order.Direction = "ASC"

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o OrderBy[Q]) Desc() OrderBy[Q] {
	order := o()
	order.Direction = "DESC"

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o OrderBy[Q]) Collate(collation string) OrderBy[Q] {
	order := o()
	order.CollationName = collation

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

type CTEChain[Q interface{ AppendWith(clause.CTE) }] func() clause.CTE

func (c CTEChain[Q]) Apply(q Q) {
	q.AppendWith(c())
}

func (c CTEChain[Q]) As(q bob.Query) CTEChain[Q] {
	cte := c()
	cte.Query = q
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

type LockChain[Q interface{ SetFor(clause.For) }] func() clause.For

func (l LockChain[Q]) Apply(q Q) {
	q.SetFor(l())
}

func (l LockChain[Q]) NoWait() LockChain[Q] {
	lock := l()
	lock.Wait = clause.LockWaitNoWait
	return LockChain[Q](func() clause.For {
		return lock
	})
}

func (l LockChain[Q]) SkipLocked() LockChain[Q] {
	lock := l()
	lock.Wait = clause.LockWaitSkipLocked
	return LockChain[Q](func() clause.For {
		return lock
	})
}

type WindowMod[Q interface{ AppendWindow(clause.NamedWindow) }] struct {
	Name string
	*WindowChain[*WindowMod[Q]]
}

func (w WindowMod[Q]) Apply(q Q) {
	q.AppendWindow(clause.NamedWindow{
		Name:       w.Name,
		Definition: w.def,
	})
}

type WindowChain[T any] struct {
	def  clause.WindowDef
	Wrap T
}

func (w *WindowChain[T]) From(name string) T {
	w.def.SetFrom(name)
	return w.Wrap
}

func (w *WindowChain[T]) PartitionBy(condition ...any) T {
	w.def.AddPartitionBy(condition...)
	return w.Wrap
}

func (w *WindowChain[T]) OrderBy(order ...any) T {
	w.def.AddOrderBy(order...)
	return w.Wrap
}

func (w *WindowChain[T]) Range() T {
	w.def.SetMode("RANGE")
	return w.Wrap
}

func (w *WindowChain[T]) Rows() T {
	w.def.SetMode("ROWS")
	return w.Wrap
}

func (w *WindowChain[T]) FromUnboundedPreceding() T {
	w.def.SetStart("UNBOUNDED PRECEDING")
	return w.Wrap
}

func (w *WindowChain[T]) FromPreceding(exp any) T {
	w.def.SetStart(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) ([]any, error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " PRECEDING")
		}),
	)
	return w.Wrap
}

func (w *WindowChain[T]) FromCurrentRow() T {
	w.def.SetStart("CURRENT ROW")
	return w.Wrap
}

func (w *WindowChain[T]) FromFollowing(exp any) T {
	w.def.SetStart(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) ([]any, error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " FOLLOWING")
		}),
	)
	return w.Wrap
}

func (w *WindowChain[T]) ToPreceding(exp any) T {
	w.def.SetEnd(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) ([]any, error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " PRECEDING")
		}),
	)
	return w.Wrap
}

func (w *WindowChain[T]) ToCurrentRow(count int) T {
	w.def.SetEnd("CURRENT ROW")
	return w.Wrap
}

func (w *WindowChain[T]) ToFollowing(exp any) T {
	w.def.SetEnd(bob.ExpressionFunc(
		func(w io.Writer, d bob.Dialect, start int) ([]any, error) {
			return bob.ExpressIf(w, d, start, exp, true, "", " FOLLOWING")
		}),
	)
	return w.Wrap
}

func (w *WindowChain[T]) ToUnboundedFollowing() T {
	w.def.SetEnd("UNBOUNDED FOLLOWING")
	return w.Wrap
}
