package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
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
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.TableRef = clause.TableRef{
			Expression: name,
			Columns:    columns,
		}
	})
}

func IntoAs(name any, alias string, columns ...string) bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.TableRef = clause.TableRef{
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
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Query = q
	})
}

func OnConflict(columns ...any) mods.Conflict[*dialect.InsertQuery] {
	return mods.Conflict[*dialect.InsertQuery](func() clause.ConflictClause {
		return clause.ConflictClause{
			Target: clause.ConflictTarget{
				Columns: columns,
			},
		}
	})
}

func Returning(clauses ...any) bob.Mod[*dialect.InsertQuery] {
	return mods.Returning[*dialect.InsertQuery](clauses)
}

//========================================
// For use in ON CONFLICT DO UPDATE SET
//========================================

func Set(sets ...bob.Expression) bob.Mod[*clause.ConflictClause] {
	return bob.ModFunc[*clause.ConflictClause](func(c *clause.ConflictClause) {
		c.Set.Set = append(c.Set.Set, internal.ToAnySlice(sets)...)
	})
}

func SetCol(from string) mods.Set[*clause.ConflictClause] {
	return mods.Set[*clause.ConflictClause]{from}
}

func SetExcluded(cols ...string) bob.Mod[*clause.ConflictClause] {
	exprs := make([]any, 0, len(cols))
	for _, col := range cols {
		if col == "" {
			continue
		}
		exprs = append(exprs,
			expr.Join{Exprs: []bob.Expression{
				expr.Quote(col), expr.Raw("= EXCLUDED."), expr.Quote(col),
			}},
		)
	}

	return bob.ModFunc[*clause.ConflictClause](func(c *clause.ConflictClause) {
		c.Set.Set = append(c.Set.Set, exprs...)
	})
}

func Where(e bob.Expression) bob.Mod[*clause.ConflictClause] {
	return bob.ModFunc[*clause.ConflictClause](func(c *clause.ConflictClause) {
		c.Where.Conditions = append(c.Where.Conditions, e)
	})
}
