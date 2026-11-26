package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[dialect.DeleteEditor] {
	return dialect.With[dialect.DeleteEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.DeleteEditor] {
	return mods.Recursive[dialect.DeleteEditor](r)
}

func From(name any) dialect.FromChain[dialect.DeleteEditor] {
	return dialect.From[dialect.DeleteEditor](name)
}

func IndexedBy(i string) bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(q dialect.DeleteEditor) {
		q.SetIndexedBy(&i)
	})
}

func NotIndexed() bob.Mod[dialect.DeleteEditor] {
	return bob.ModFunc[dialect.DeleteEditor](func(q dialect.DeleteEditor) {
		var s string
		q.SetIndexedBy(&s)
	})
}

func Where(e bob.Expression) mods.Where[dialect.DeleteEditor] {
	return mods.Where[dialect.DeleteEditor]{E: e}
}

func Returning(clauses ...any) bob.Mod[dialect.DeleteEditor] {
	return mods.Returning[dialect.DeleteEditor](clauses)
}

func Limit(count any) bob.Mod[dialect.DeleteEditor] {
	return mods.Limit[dialect.DeleteEditor]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[dialect.DeleteEditor] {
	return mods.Offset[dialect.DeleteEditor]{
		Count: count,
	}
}
