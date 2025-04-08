package dialect

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

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
	SetLateral(bool)
	AppendPartition(...string)
	AppendIndexHint(clause.IndexHint)
}

func From[Q fromable](table any) FromChain[Q] {
	return FromChain[Q](func() clause.From {
		return clause.From{
			Table: table,
		}
	})
}

type FromChain[Q fromable] func() clause.From

func (f FromChain[Q]) Apply(q Q) {
	from := f()

	q.SetTable(from.Table)
	if from.Alias != "" {
		q.SetTableAlias(from.Alias, from.Columns...)
	}

	q.SetLateral(from.Lateral)
	q.AppendPartition(from.Partitions...)
}

func (f FromChain[Q]) As(alias string, columns ...string) FromChain[Q] {
	fr := f()
	fr.Alias = alias
	fr.Columns = columns

	return FromChain[Q](func() clause.From {
		return fr
	})
}

func (f FromChain[Q]) Lateral() FromChain[Q] {
	fr := f()
	fr.Lateral = true

	return FromChain[Q](func() clause.From {
		return fr
	})
}

func (f FromChain[Q]) Partition(partitions ...string) FromChain[Q] {
	fr := f()
	fr.Partitions = append(fr.Partitions, partitions...)

	return FromChain[Q](func() clause.From {
		return fr
	})
}

func (f FromChain[Q]) index(Type, For, first string, others ...string) FromChain[Q] {
	fr := f()
	fr.IndexHints = append(fr.IndexHints, clause.IndexHint{
		Type:    Type,
		Indexes: append([]string{first}, others...),
		For:     For,
	})

	return FromChain[Q](func() clause.From {
		return fr
	})
}

func (f FromChain[Q]) UseIndex(first string, others ...string) FromChain[Q] {
	return f.index("USE", "", first, others...)
}

func (f FromChain[Q]) UseIndexForJoin(first string, others ...string) FromChain[Q] {
	return f.index("USE", "JOIN", first, others...)
}

func (f FromChain[Q]) UseIndexForOrderBy(first string, others ...string) FromChain[Q] {
	return f.index("USE", "ORDER BY", first, others...)
}

func (f FromChain[Q]) UseIndexForGroupBy(first string, others ...string) FromChain[Q] {
	return f.index("USE", "GROUP BY", first, others...)
}

func (f FromChain[Q]) IgnoreIndex(first string, others ...string) FromChain[Q] {
	return f.index("IGNORE", "", first, others...)
}

func (f FromChain[Q]) IgnoreIndexForJoin(first string, others ...string) FromChain[Q] {
	return f.index("IGNORE", "JOIN", first, others...)
}

func (f FromChain[Q]) IgnoreIndexForOrderBy(first string, others ...string) FromChain[Q] {
	return f.index("IGNORE", "ORDER BY", first, others...)
}

func (f FromChain[Q]) IgnoreIndexForGroupBy(first string, others ...string) FromChain[Q] {
	return f.index("IGNORE", "GROUP BY", first, others...)
}

func (f FromChain[Q]) ForceIndex(first string, others ...string) FromChain[Q] {
	return f.index("FORCE", "", first, others...)
}

func (f FromChain[Q]) ForceIndexForJoin(first string, others ...string) FromChain[Q] {
	return f.index("FORCE", "JOIN", first, others...)
}

func (f FromChain[Q]) ForceIndexForOrderBy(first string, others ...string) FromChain[Q] {
	return f.index("FORCE", "ORDER BY", first, others...)
}

func (f FromChain[Q]) ForceIndexForGroupBy(first string, others ...string) FromChain[Q] {
	return f.index("FORCE", "GROUP BY", first, others...)
}

func Partition[Q interface{ AppendPartition(...string) }](partitions ...string) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
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

type Joinable interface{ AppendJoin(clause.Join) }

func Join[Q Joinable](typ string, e any) JoinChain[Q] {
	return JoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: typ,
			To:   clause.From{Table: e},
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

func CrossJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.CrossJoin, e)
}

func StraightJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.StraightJoin, e)
}

type JoinChain[Q Joinable] func() clause.Join

func (j JoinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (f JoinChain[Q]) As(alias string, columns ...string) JoinChain[Q] {
	jo := f()
	jo.To.Alias = alias
	jo.To.Columns = columns

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

func (f JoinChain[Q]) Partition(partitions ...string) JoinChain[Q] {
	jo := f()
	jo.To.Partitions = append(jo.To.Partitions, partitions...)

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
