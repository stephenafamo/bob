package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*sqlite.InsertQuery] {
	return dialect.With[*sqlite.InsertQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*sqlite.InsertQuery] {
	return mods.Recursive[*sqlite.InsertQuery](r)
}

func OrAbort() bob.Mod[*sqlite.InsertQuery] {
	return dialect.OrAbort[*sqlite.InsertQuery]()
}

func OrFail() bob.Mod[*sqlite.InsertQuery] {
	return dialect.OrFail[*sqlite.InsertQuery]()
}

func OrIgnore() bob.Mod[*sqlite.InsertQuery] {
	return dialect.OrIgnore[*sqlite.InsertQuery]()
}

func OrReplace() bob.Mod[*sqlite.InsertQuery] {
	return dialect.OrReplace[*sqlite.InsertQuery]()
}

func OrRollback() bob.Mod[*sqlite.InsertQuery] {
	return dialect.OrRollback[*sqlite.InsertQuery]()
}

func Into(name any, columns ...string) bob.Mod[*sqlite.InsertQuery] {
	return mods.QueryModFunc[*sqlite.InsertQuery](func(i *sqlite.InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func IntoAs(name any, alias string, columns ...string) bob.Mod[*sqlite.InsertQuery] {
	return mods.QueryModFunc[*sqlite.InsertQuery](func(i *sqlite.InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func Values(clauses ...any) bob.Mod[*sqlite.InsertQuery] {
	return mods.Values[*sqlite.InsertQuery](clauses)
}

func Rows(rows ...[]any) bob.Mod[*sqlite.InsertQuery] {
	return mods.Rows[*sqlite.InsertQuery](rows)
}

// Insert from a query
func Query(q bob.Query) bob.Mod[*sqlite.InsertQuery] {
	return mods.QueryModFunc[*sqlite.InsertQuery](func(i *sqlite.InsertQuery) {
		i.Query = q
	})
}

func OnConflict(column any, where ...any) mods.Conflict[*sqlite.InsertQuery] {
	if column != nil {
		column = sqlite.P(column)
	}
	return mods.Conflict[*sqlite.InsertQuery](func() clause.Conflict {
		return clause.Conflict{
			Target: clause.ConflictTarget{
				Target: column,
				Where:  where,
			},
		}
	})
}

func Returning(clauses ...any) bob.Mod[*sqlite.InsertQuery] {
	return mods.Returning[*sqlite.InsertQuery](clauses)
}
