package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

func Into(name any, columns ...string) bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetTable(name)
		i.SetTableColumns(columns...)
	})
}

func LowPriority() bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func HighPriority() bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.AppendModifier("HIGH_PRIORITY")
	})
}

func Ignore() bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.AppendModifier("IGNORE")
	})
}

func Partition(partitions ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.Partition[dialect.InsertEditor](partitions...)
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

// Insert with Set a = b
func Set(col string, val any) bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.AppendAssignment(dialect.Set{
			Col: col,
			Val: val,
		})
	})
}

func As(rowAlias string, colAlias ...string) bob.Mod[dialect.InsertEditor] {
	return bob.ModFunc[dialect.InsertEditor](func(i dialect.InsertEditor) {
		i.SetRowAlias(rowAlias)
		i.SetColumnAliases(colAlias...)
	})
}

func OnDuplicateKeyUpdate(clauses ...bob.Mod[*clause.Set]) bob.Mod[dialect.InsertEditor] {
	sets := clause.Set{}
	for _, m := range clauses {
		m.Edit(&sets)
	}

	return bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) {
		q.AppendDuplicateKeyUpdate(sets.Set...)
	})
}

//========================================
// For use in ON DUPLICATE KEY UPDATE
//========================================

func Update(exprs ...bob.Expression) bob.Mod[*clause.Set] {
	return bob.ModFunc[*clause.Set](func(c *clause.Set) {
		c.Set = append(c.Set, internal.ToAnySlice(exprs)...)
	})
}

// UpdateCol sets one column in ON DUPLICATE KEY UPDATE. The column name is quoted automatically.
func UpdateCol(col string) mods.Set[*clause.Set] {
	return mods.Set[*clause.Set]{Col: expr.Quote(col)}
}

func UpdateWithAlias(alias string, cols ...string) bob.Mod[*clause.Set] {
	newCols := make([]any, len(cols))
	for i, c := range cols {
		newCols[i] = dialect.Set{Col: c, Val: expr.Quote(alias, c)}
	}

	return bob.ModFunc[*clause.Set](func(s *clause.Set) {
		s.Set = append(s.Set, newCols...)
	})
}

func UpdateWithValues(cols ...string) bob.Mod[*clause.Set] {
	newCols := make([]any, len(cols))
	for i, c := range cols {
		newCols[i] = dialect.Set{
			Col: c,
			Val: dialect.NewFunction("VALUES", expr.Quote(c)),
		}
	}

	return bob.ModFunc[*clause.Set](func(s *clause.Set) {
		s.Set = append(s.Set, newCols...)
	})
}
