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

	q.SetLateral(from.Lateral)
}

func (f FromChain[Q]) As(alias string, columns ...string) FromChain[Q] {
	fr := f()
	fr.Alias = alias
	fr.Columns = columns

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

type OrderCombined OrderBy[*SelectQuery]

func (o OrderCombined) Apply(q *SelectQuery) {
	q.CombinedOrder.AppendOrder(o())
}
