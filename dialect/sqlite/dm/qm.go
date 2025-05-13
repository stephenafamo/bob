package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.DeleteQuery] {
	return dialect.With[*dialect.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.DeleteQuery] {
	return mods.Recursive[*dialect.DeleteQuery](r)
}

func From(name any) dialect.FromChain[*dialect.DeleteQuery] {
	return dialect.From[*dialect.DeleteQuery](name)
}

func IndexedBy(i string) bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		q.TableRef.IndexedBy = &i
	})
}

func NotIndexed() bob.Mod[*dialect.DeleteQuery] {
	return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
		var s string
		q.TableRef.IndexedBy = &s
	})
}

func Where(e bob.Expression) mods.Where[*dialect.DeleteQuery] {
	return mods.Where[*dialect.DeleteQuery]{E: e}
}

func Returning(clauses ...any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Returning[*dialect.DeleteQuery](clauses)
}

func Limit(count any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Limit[*dialect.DeleteQuery]{
		Count: count,
	}
}

func Offset(count any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Offset[*dialect.DeleteQuery]{
		Count: count,
	}
}
