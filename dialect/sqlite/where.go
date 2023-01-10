package sqlite

import (
	"github.com/stephenafamo/bob"
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

type WhereMod[Q Filterable, C any] struct {
	name Expression
}

func (w WhereMod[Q, C]) EQ(val C) bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).EQ(Arg(val))}
}

func (w WhereMod[Q, C]) NE(val C) bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).NE(Arg(val))}
}

func (w WhereMod[Q, C]) LT(val C) bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).LT(Arg(val))}
}

func (w WhereMod[Q, C]) LTE(val C) bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).LTE(Arg(val))}
}

func (w WhereMod[Q, C]) GT(val C) bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).GT(Arg(val))}
}

func (w WhereMod[Q, C]) GTE(val C) bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).GTE(Arg(val))}
}

func (w WhereMod[Q, C]) In(slice ...C) bob.Mod[Q] {
	values := make([]any, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return mods.Where[Q]{X(w.name).In(Arg(values...))}
}

func (w WhereMod[Q, C]) NotIn(slice ...C) bob.Mod[Q] {
	values := make([]any, 0, len(slice))
	for _, value := range slice {
		values = append(values, value)
	}
	return mods.Where[Q]{X(w.name).NotIn(Arg(values...))}
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

func (w WhereNullMod[Q, C]) IsNull() bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).IsNull()}
}

func (w WhereNullMod[Q, C]) IsNotNull() bob.Mod[Q] {
	return mods.Where[Q]{X(w.name).IsNotNull()}
}
