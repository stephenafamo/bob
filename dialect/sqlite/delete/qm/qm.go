package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*sqlite.DeleteQuery] {
	return dialect.With[*sqlite.DeleteQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*sqlite.DeleteQuery] {
	return mods.Recursive[*sqlite.DeleteQuery](r)
}

func From(name any) dialect.FromChain[*sqlite.DeleteQuery] {
	return dialect.From[*sqlite.DeleteQuery](name)
}

func Where(e bob.Expression) bob.Mod[*sqlite.DeleteQuery] {
	return mods.Where[*sqlite.DeleteQuery]{e}
}

func WhereClause(clause string, args ...any) bob.Mod[*sqlite.DeleteQuery] {
	return mods.Where[*sqlite.DeleteQuery]{sqlite.Raw(clause, args...)}
}

func Returning(clauses ...any) bob.Mod[*sqlite.DeleteQuery] {
	return mods.Returning[*sqlite.DeleteQuery](clauses)
}
