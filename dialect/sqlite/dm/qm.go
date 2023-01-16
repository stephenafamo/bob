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

func Where(e any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Where[*dialect.DeleteQuery]{e}
}

func Returning(clauses ...any) bob.Mod[*dialect.DeleteQuery] {
	return mods.Returning[*dialect.DeleteQuery](clauses)
}
