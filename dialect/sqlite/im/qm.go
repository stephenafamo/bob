package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/mods"
)

func With(name string, columns ...string) dialect.CTEChain[*dialect.InsertQuery] {
	return dialect.With[*dialect.InsertQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.InsertQuery] {
	return mods.Recursive[*dialect.InsertQuery](r)
}

func OrAbort() bob.Mod[*dialect.InsertQuery] {
	return dialect.OrAbort[*dialect.InsertQuery]()
}

func OrFail() bob.Mod[*dialect.InsertQuery] {
	return dialect.OrFail[*dialect.InsertQuery]()
}

func OrIgnore() bob.Mod[*dialect.InsertQuery] {
	return dialect.OrIgnore[*dialect.InsertQuery]()
}

func OrReplace() bob.Mod[*dialect.InsertQuery] {
	return dialect.OrReplace[*dialect.InsertQuery]()
}

func OrRollback() bob.Mod[*dialect.InsertQuery] {
	return dialect.OrRollback[*dialect.InsertQuery]()
}

func Into(name any, columns ...string) bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Columns:    columns,
		}
	})
}

func IntoAs(name any, alias string, columns ...string) bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Table = clause.Table{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		}
	})
}

func Values(clauses ...bob.Expression) bob.Mod[*dialect.InsertQuery] {
	return mods.Values[*dialect.InsertQuery](clauses)
}

func Rows(rows ...[]bob.Expression) bob.Mod[*dialect.InsertQuery] {
	return mods.Rows[*dialect.InsertQuery](rows)
}

// Insert from a query
func Query(q bob.Query) bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Query = q
	})
}

func OnConflict(columns ...any) mods.Conflict[*dialect.InsertQuery] {
	return mods.Conflict[*dialect.InsertQuery](func() clause.Conflict {
		return clause.Conflict{
			Target: clause.ConflictTarget{
				Columns: columns,
			},
		}
	})
}

func Returning(clauses ...any) bob.Mod[*dialect.InsertQuery] {
	return mods.Returning[*dialect.InsertQuery](clauses)
}
