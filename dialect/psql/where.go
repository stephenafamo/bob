package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

type mod[Q interface{ AppendWhere(e ...any) }] struct {
	e bob.Expression
}

func (d mod[Q]) Apply(q Q) {
	q.AppendWhere(d.e)
}

type Filterable interface {
	AppendWhere(...any)
}

func Where[Q Filterable, C any](name Expression) WhereMod[Q, C] {
	return WhereMod[Q, C]{
		name: name,
	}
}

func WhereOr[Q Filterable](whereMods ...mod[Q]) mod[Q] {
	exprs := make([]bob.Expression, len(whereMods))
	for i, mod := range whereMods {
		exprs[i] = mod.e
	}

	return mod[Q]{
		e: Or(exprs...),
	}
}

func WhereAnd[Q Filterable](whereMods ...mod[Q]) mod[Q] {
	exprs := make([]bob.Expression, len(whereMods))
	for i, mod := range whereMods {
		exprs[i] = mod.e
	}

	return mod[Q]{
		e: And(exprs...),
	}
}

type WhereMod[Q Filterable, C any] struct {
	name Expression
}

func (w WhereMod[Q, C]) EQ(val C) mod[Q] {
	return mod[Q]{w.name.EQ(Arg(val))}
}

func (w WhereMod[Q, C]) NE(val C) mod[Q] {
	return mod[Q]{w.name.NE(Arg(val))}
}

func (w WhereMod[Q, C]) LT(val C) mod[Q] {
	return mod[Q]{w.name.LT(Arg(val))}
}

func (w WhereMod[Q, C]) LTE(val C) mod[Q] {
	return mod[Q]{w.name.LTE(Arg(val))}
}

func (w WhereMod[Q, C]) GT(val C) mod[Q] {
	return mod[Q]{w.name.GT(Arg(val))}
}

func (w WhereMod[Q, C]) GTE(val C) mod[Q] {
	return mod[Q]{w.name.GTE(Arg(val))}
}

func (w WhereMod[Q, C]) In(slice ...C) mod[Q] {
	values := make([]any, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return mod[Q]{w.name.In(Arg(values...))}
}

func (w WhereMod[Q, C]) NotIn(slice ...C) mod[Q] {
	values := make([]any, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return mod[Q]{w.name.NotIn(Arg(values...))}
}

func (w WhereMod[Q, C]) Like(val C) mod[Q] {
	return mod[Q]{w.name.Like(Arg(val))}
}

func (w WhereMod[Q, C]) ILike(val C) mod[Q] {
	return mod[Q]{expr.OP("ILIKE", w.name, Arg(val))}
}

func WhereNull[Q Filterable, C any](name Expression) WhereNullMod[Q, C] {
	return WhereNullMod[Q, C]{
		WhereMod: Where[Q, C](name),
	}
}

type WhereNullMod[Q interface {
	AppendWhere(e ...any)
}, C any] struct {
	WhereMod[Q, C]
}

func (w WhereNullMod[Q, C]) IsNull() mod[Q] {
	return mod[Q]{w.name.IsNull()}
}

func (w WhereNullMod[Q, C]) IsNotNull() mod[Q] {
	return mod[Q]{w.name.IsNotNull()}
}
