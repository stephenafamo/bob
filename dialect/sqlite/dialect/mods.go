package dialect

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/clause"
	"github.com/twitter-payments/bob/expr"
	"github.com/twitter-payments/bob/mods"
)

func With[Q interface{ AppendWith(clause.CTE) }](name string, columns ...string) CTEChain[Q] {
	return CTEChain[Q](func() clause.CTE {
		return clause.CTE{
			Name:    name,
			Columns: columns,
		}
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

func OrAbort[Q interface{ SetOr(string) }]() bob.Mod[Q] {
	return bob.ModFunc[Q](func(i Q) {
		i.SetOr("ABORT")
	})
}

func OrFail[Q interface{ SetOr(string) }]() bob.Mod[Q] {
	return bob.ModFunc[Q](func(i Q) {
		i.SetOr("FAIL")
	})
}

func OrIgnore[Q interface{ SetOr(string) }]() bob.Mod[Q] {
	return bob.ModFunc[Q](func(i Q) {
		i.SetOr("IGNORE")
	})
}

func OrReplace[Q interface{ SetOr(string) }]() bob.Mod[Q] {
	return bob.ModFunc[Q](func(i Q) {
		i.SetOr("REPLACE")
	})
}

func OrRollback[Q interface{ SetOr(string) }]() bob.Mod[Q] {
	return bob.ModFunc[Q](func(i Q) {
		i.SetOr("ROLLBACK")
	})
}

type fromable interface {
	SetTable(any)
	SetTableAlias(alias string, columns ...string)
	SetIndexedBy(*string)
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

	q.SetIndexedBy(from.IndexedBy)
}

func (f FromChain[Q]) As(alias string) FromChain[Q] {
	fr := f()
	fr.Alias = alias

	return FromChain[Q](func() clause.From {
		return fr
	})
}

func (f FromChain[Q]) NotIndexed() bob.Mod[Q] {
	i := ""
	fr := f()
	fr.SetIndexedBy(&i)

	return FromChain[Q](func() clause.From {
		return fr
	})
}

func (f FromChain[Q]) IndexedBy(indexName string) bob.Mod[Q] {
	fr := f()
	fr.SetIndexedBy(&indexName)

	return FromChain[Q](func() clause.From {
		return fr
	})
}

type JoinChain[Q interface{ AppendJoin(clause.Join) }] func() clause.Join

func (j JoinChain[Q]) Apply(q Q) {
	q.AppendJoin(j())
}

func (f JoinChain[Q]) NotIndexed() bob.Mod[Q] {
	i := ""
	jo := f()
	jo.To.SetIndexedBy(&i)

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (f JoinChain[Q]) IndexedBy(indexName string) bob.Mod[Q] {
	jo := f()
	jo.To.SetIndexedBy(&indexName)

	return JoinChain[Q](func() clause.Join {
		return jo
	})
}

func (j JoinChain[Q]) As(alias string) JoinChain[Q] {
	jo := j()
	jo.To.Alias = alias

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

func FullJoin[Q Joinable](e any) JoinChain[Q] {
	return Join[Q](clause.FullJoin, e)
}

func CrossJoin[Q Joinable](e any) CrossJoinChain[Q] {
	return CrossJoinChain[Q](func() clause.Join {
		return clause.Join{
			Type: clause.CrossJoin,
			To:   clause.From{Table: e},
		}
	})
}

type OrderBy[Q interface{ AppendOrder(clause.OrderDef) }] func() clause.OrderDef

func (s OrderBy[Q]) Apply(q Q) {
	q.AppendOrder(s())
}

func (o OrderBy[Q]) Collate(collationName string) OrderBy[Q] {
	order := o()
	order.Collation = collationName

	return OrderBy[Q](func() clause.OrderDef {
		return order
	})
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
