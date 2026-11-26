package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

// With starts a CTE. The name and column list are quoted as SQL identifiers.
func With(name string, columns ...string) dialect.CTEChain[dialect.InsertEditor] {
	return dialect.With[dialect.InsertEditor](name, columns...)
}

func Recursive(r bool) bob.Mod[dialect.InsertEditor] {
	return mods.Recursive[dialect.InsertEditor](r)
}

// Into sets the insert target. Use psql.Quote(...) for quoted table names; column names
// in columns are quoted automatically.
func Into(name any, columns ...string) bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetTableRef(clause.TableRef{
			Expression: name,
			Columns:    columns,
		})
	})
}

func IntoAs(name any, alias string, columns ...string) bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetTableRef(clause.TableRef{
			Expression: name,
			Alias:      alias,
			Columns:    columns,
		})
	})
}

func OverridingSystem() bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetOverriding(dialect.OverridingSystem)
	})
}

func OverridingUser() bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetOverriding(dialect.OverridingUser)
	})
}

func Values(clauses ...bob.Expression) bob.Mod[dialect.InsertEditor] {
	return mods.Values[dialect.InsertEditor](clauses)
}

func Rows(rows ...[]bob.Expression) bob.Mod[dialect.InsertEditor] {
	return mods.BatchRows[dialect.InsertEditor](rows)
}

// Insert from a query
func Query(q bob.Query) bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetInsertQuery(q)
	})
}

// OnConflict starts an ON CONFLICT clause. Pass conflict target columns as
// psql.Quote(...) or another Expression; strings are written as raw SQL.
func OnConflict(columns ...any) mods.Conflict[dialect.InsertEditor] {
	return mods.ConflictColumns[dialect.InsertEditor](columns...)
}

func OnConflictOnConstraint(constraint string) mods.Conflict[dialect.InsertEditor] {
	return mods.ConflictOnConstraint[dialect.InsertEditor](constraint)
}

func Returning(clauses ...any) mods.Returning[dialect.InsertEditor] {
	return mods.Returning[dialect.InsertEditor](clauses)
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

// SetExpr is like SetCol but the column LHS is any expression (e.g. psql.Quote("t", "col")).
func SetExpr(col bob.Expression) mods.Set[*clause.ConflictClause] {
	return mods.Set[*clause.ConflictClause]{Col: col}
}

// SetCols creates a multi-column setter: (columns...) = ROW(...) | (values...) | (subquery)
func SetCols(columns ...string) clause.SetCols[*clause.ConflictClause] {
	return clause.NewSetCols[*clause.ConflictClause](columns...).Options(clause.SetColsOptions{RowPrefix: "ROW"})
}

// Excluded references a column from the EXCLUDED pseudo-table in ON CONFLICT DO UPDATE.
//
//	SQL: EXCLUDED."col"
//	Go: im.Excluded("col")
func Excluded(column string) dialect.Expression {
	return dialect.NewExpression(
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
