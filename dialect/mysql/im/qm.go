package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

func Into(name any, columns ...string) bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Table = name
		i.Columns = columns
	})
}

func LowPriority() bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func HighPriority() bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.AppendModifier("HIGH_PRIORITY")
	})
}

func Ignore() bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.AppendModifier("IGNORE")
	})
}

func Partition(partitions ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.Partition[*dialect.InsertQuery](partitions...)
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

// Insert with Set a = b
func Set(col string, val any) bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Sets = append(i.Sets, dialect.Set{
			Col: col,
			Val: val,
		})
	})
}

func As(rowAlias string, colAlias ...string) bob.Mod[*dialect.InsertQuery] {
	return bob.ModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.RowAlias = rowAlias
		i.ColumnAlias = colAlias
	})
}

func OnDuplicateKeyUpdate(clauses ...bob.Mod[*clause.Set]) bob.Mod[*dialect.InsertQuery] {
	sets := clause.Set{}
	for _, m := range clauses {
		m.Apply(&sets)
	}

	return bob.ModFunc[*dialect.InsertQuery](func(q *dialect.InsertQuery) {
		q.DuplicateKeyUpdate.Set = append(q.DuplicateKeyUpdate.Set, sets.Set...)
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

func UpdateCol(col string) mods.Set[*clause.Set] {
	return mods.Set[*clause.Set]{col}
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
