package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

type Distinct struct {
	On []any
}

func (di Distinct) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("DISTINCT")
	return bob.ExpressSlice(ctx, w, d, start, di.On, " ON (", ", ", ")")
}

func With[Q interface{ AppendCTE(bob.Expression) }](name string, columns ...string) CTEChain[Q] {
	return CTEChain[Q](func() clause.CTE {
		return clause.CTE{
			Name:    name,
			Columns: columns,
		}
	})
}

type fromable interface {
	SetTable(any)
	SetTableAlias(alias string, columns ...string)
	SetOnly(bool)
	SetLateral(bool)
	SetWithOrdinality(bool)
}

func From[Q fromable](table any) FromChain[Q] {
	return FromChain[Q](func() clause.TableRef {
		return clause.TableRef{Expression: table}
	})
}

type FromChain[Q fromable] func() clause.TableRef

func (f FromChain[Q]) Apply(q Q) {
	from := f()

	q.SetTable(from.Expression)
	if from.Alias != "" {
		q.SetTableAlias(from.Alias, from.Columns...)
	}

	q.SetOnly(from.Only)
	q.SetLateral(from.Lateral)
	q.SetWithOrdinality(from.WithOrdinality)
}

func (f FromChain[Q]) As(alias string, columns ...string) FromChain[Q] {
	fr := f()
	fr.Alias = alias
	fr.Columns = columns

	return FromChain[Q](func() clause.TableRef {
		return fr
	})
}

func (f FromChain[Q]) Only() FromChain[Q] {
	fr := f()
	fr.Only = true

	return FromChain[Q](func() clause.TableRef {
		return fr
	})
}

func (f FromChain[Q]) Lateral() FromChain[Q] {
	fr := f()
	fr.Lateral = true

	return FromChain[Q](func() clause.TableRef {
		return fr
	})
}

func (f FromChain[Q]) WithOrdinality() FromChain[Q] {
	fr := f()
	fr.WithOrdinality = true

	return FromChain[Q](func() clause.TableRef {
		return fr
	})
}

type Joinable interface{ AppendJoin(clause.Join) }

func Join[Q Joinable](typ string, e any) JoinChain[Q] {
	return JoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: typ,
			To:   clause.TableRef{Expression: e},
		}
	})
}

func InnerJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.InnerJoin, e)
}

func LeftJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.LeftJoin, e)
}

func RightJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.RightJoin, e)
}

func FullJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.FullJoin, e)
}

func CrossJoin[Q Joinable](e any) CrossJoinChain[Q] {
	return CrossJoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.CrossJoin,
			To:   clause.TableRef{Expression: e},
		}
	})
}

type JoinChain[Q Joinable] func() clause.Join

func (j JoinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (j JoinChain[Q]) As(alias string, columns ...string) JoinChain[Q] {
	jo := j()
	jo.To.Alias = alias
	jo.To.Columns = columns

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (f JoinChain[Q]) Only() JoinChain[Q] {
	jo := f()
	jo.To.Only = true

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (f JoinChain[Q]) Lateral() JoinChain[Q] {
	jo := f()
	jo.To.Lateral = true

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (f JoinChain[Q]) WithOrdinality() JoinChain[Q] {
	jo := f()
	jo.To.WithOrdinality = true

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (j JoinChain[Q]) Natural() bob.Mod[Q] {
	jo := j()
	jo.Natural = true

	return mods.Join[Q](jo)
}

func (j JoinChain[Q]) On(on ...bob.Expression) bob.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, on...)

	return mods.Join[Q](jo)
}

func (j JoinChain[Q]) OnEQ(a, b bob.Expression) bob.Mod[Q] {
	jo := j()
	jo.On = append(jo.On, expr.X[Expression, Expression](a).EQ(b))

	return mods.Join[Q](jo)
}

func (j JoinChain[Q]) Using(using ...string) bob.Mod[Q] {
	jo := j()
	jo.Using = using

	return mods.Join[Q](jo)
}

type CrossJoinChain[Q Joinable] func() clause.Join

func (j CrossJoinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (j CrossJoinChain[Q]) As(alias string, columns ...string) bob.Mod[Q] {
	jo := j()
	jo.To.Alias = alias
	jo.To.Columns = columns

	return CrossJoinChain[Q](func() clause.Join {
		return jo
	})
}

type OrderCombined OrderBy[*SelectQuery]

func (o OrderCombined) Apply(q *SelectQuery) {
	q.CombinedOrder.AppendOrder(o())
}

type OrderBy[Q interface{ AppendOrder(bob.Expression) }] func() clause.OrderDef

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

func (o OrderBy[Q]) Using(operator string) OrderBy[Q] {
	order := o()
	order.Direction = "USING " + operator

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o OrderBy[Q]) NullsFirst() OrderBy[Q] {
	order := o()
	order.Nulls = "FIRST"

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o OrderBy[Q]) NullsLast() OrderBy[Q] {
	order := o()
	order.Nulls = "LAST"

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

func (o OrderBy[Q]) Collate(collationName string) OrderBy[Q] {
	order := o()
	order.Collation = collationName

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
}

type CTEChain[Q interface{ AppendCTE(bob.Expression) }] func() clause.CTE

func (c CTEChain[Q]) Apply(q Q) {
	q.AppendCTE(c())
}

func (c CTEChain[Q]) As(q bob.Query) CTEChain[Q] {
	cte := c()
	cte.Query = q
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

func (c CTEChain[Q]) NotMaterialized() CTEChain[Q] {
	b := false
	cte := c()
	cte.Materialized = &b
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

func (c CTEChain[Q]) Materialized() CTEChain[Q] {
	b := true
	cte := c()
	cte.Materialized = &b
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

func (c CTEChain[Q]) SearchBreadth(setCol string, searchCols ...string) CTEChain[Q] {
	cte := c()
	cte.Search = clause.CTESearch{
		Order:   clause.SearchDepth,
		Columns: searchCols,
		Set:     setCol,
	}
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

func (c CTEChain[Q]) SearchDepth(setCol string, searchCols ...string) CTEChain[Q] {
	cte := c()
	cte.Search = clause.CTESearch{
		Order:   clause.SearchDepth,
		Columns: searchCols,
		Set:     setCol,
	}
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

func (c CTEChain[Q]) Cycle(set, using string, cols ...string) CTEChain[Q] {
	cte := c()
	cte.Cycle.Set = set
	cte.Cycle.Using = using
	cte.Cycle.Columns = cols
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

func (c CTEChain[Q]) CycleValue(value, defaultVal any) CTEChain[Q] {
	cte := c()
	cte.Cycle.SetVal = value
	cte.Cycle.DefaultVal = defaultVal
	return CTEChain[Q](func() clause.CTE {
		return cte
	})
}

type LockChain[Q interface{ AppendLock(bob.Expression) }] func() clause.Lock

func (l LockChain[Q]) Apply(q Q) {
	q.AppendLock(l())
}

func (l LockChain[Q]) NoWait() LockChain[Q] {
	lock := l()
	lock.Wait = clause.LockWaitNoWait
	return LockChain[Q](func() clause.Lock {
		return lock
	})
}

func (l LockChain[Q]) SkipLocked() LockChain[Q] {
	lock := l()
	lock.Wait = clause.LockWaitSkipLocked
	return LockChain[Q](func() clause.Lock {
		return lock
	})
}
