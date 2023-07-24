package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

type Filterable interface {
	AppendWhere(...any)
}

func Where[Q Filterable, C any](name Expression) WhereMod[Q, C] {
	return WhereMod[Q, C]{
		name: name,
	}
}

func WhereOr[Q Filterable](whereMods ...mods.Where[Q]) mods.Where[Q] {
	exprs := make([]bob.Expression, len(whereMods))
	for i, mod := range whereMods {
		exprs[i] = mod.E
	}

	return mods.Where[Q]{
		E: Or(exprs...),
	}
}

func WhereAnd[Q Filterable](whereMods ...mods.Where[Q]) mods.Where[Q] {
	exprs := make([]bob.Expression, len(whereMods))
	for i, mod := range whereMods {
		exprs[i] = mod.E
	}

	return mods.Where[Q]{
		E: And(exprs...),
	}
}

type WhereMod[Q Filterable, C any] struct {
	name Expression
}

func (w WhereMod[Q, C]) EQ(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.EQ(Arg(val))}
}

func (w WhereMod[Q, C]) NE(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.NE(Arg(val))}
}

func (w WhereMod[Q, C]) LT(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.LT(Arg(val))}
}

func (w WhereMod[Q, C]) LTE(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.LTE(Arg(val))}
}

func (w WhereMod[Q, C]) GT(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.GT(Arg(val))}
}

func (w WhereMod[Q, C]) GTE(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.GTE(Arg(val))}
}

func (w WhereMod[Q, C]) In(slice ...C) mods.Where[Q] {
	values := make([]any, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return mods.Where[Q]{E: w.name.In(Arg(values...))}
}

func (w WhereMod[Q, C]) NotIn(slice ...C) mods.Where[Q] {
	values := make([]any, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return mods.Where[Q]{E: w.name.NotIn(Arg(values...))}
}

func (w WhereMod[Q, C]) Like(val C) mods.Where[Q] {
	return mods.Where[Q]{E: w.name.Like(Arg(val))}
}

func (w WhereMod[Q, C]) ILike(val C) mods.Where[Q] {
	return mods.Where[Q]{E: expr.OP("ILIKE", w.name, Arg(val))}
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

func (w WhereNullMod[Q, C]) IsNull() mods.Where[Q] {
	return mods.Where[Q]{E: w.name.IsNull()}
}

func (w WhereNullMod[Q, C]) IsNotNull() mods.Where[Q] {
	return mods.Where[Q]{E: w.name.IsNotNull()}
}
