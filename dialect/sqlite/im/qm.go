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
	return mods.ConflictColumns[*dialect.InsertQuery](columns...)
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

// SetCol sets one column in ON CONFLICT DO UPDATE SET. The column name is quoted automatically.
// For qualified names or other expressions on the LHS, use SetExpr.
func SetCol(from string) mods.Set[*clause.ConflictClause] {
	return mods.Set[*clause.ConflictClause]{Col: expr.Quote(from)}
}

// SetExpr is like SetCol but the column LHS is any expression (e.g. sqlite.Quote("t", "col")).
func SetExpr(col bob.Expression) mods.Set[*clause.ConflictClause] {
	return mods.Set[*clause.ConflictClause]{Col: col}
}

// SetCols creates a multi-column setter: (columns...) = (values...) | (subquery)
func SetCols(columns ...string) clause.SetCols[*clause.ConflictClause] {
	return clause.NewSetCols[*clause.ConflictClause](columns...)
}

// Excluded references a column from the EXCLUDED pseudo-table in ON CONFLICT DO UPDATE.
func Excluded(column string) dialect.Expression {
	return dialect.Expression{}.New(
		expr.Join{
			Exprs: []bob.Expression{expr.Raw("EXCLUDED."), expr.Quote(column)},
			Sep:   expr.NoSep,
		},
	)
}

func SetExcluded(cols ...string) bob.Mod[*clause.ConflictClause] {
	exprs := make([]bob.Expression, 0, len(cols))
	for _, col := range cols {
		if col == "" {
			continue
		}
		exprs = append(exprs, expr.OP("=", expr.Quote(col), Excluded(col)))
	}

	return Set(exprs...)
}

func Where(e bob.Expression) bob.Mod[*clause.ConflictClause] {
	return bob.ModFunc[*clause.ConflictClause](func(c *clause.ConflictClause) {
		c.Where.Conditions = append(c.Where.Conditions, e)
	})
}
