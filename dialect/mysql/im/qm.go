package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func Into(name any, columns ...string) bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Table = name
		i.Columns = columns
	})
}

func LowPriority() bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func HighPriority() bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.AppendModifier("HIGH_PRIORITY")
	})
}

func Ignore() bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
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
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Query = q
	})
}

// Insert with Set a = b
func Set(col string, val any) bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.Sets = append(i.Sets, dialect.Set{
			Col: col,
			Val: val,
		})
	})
}

func As(rowAlias string, colAlias ...string) bob.Mod[*dialect.InsertQuery] {
	return mods.QueryModFunc[*dialect.InsertQuery](func(i *dialect.InsertQuery) {
		i.RowAlias = rowAlias
		i.ColumnAlias = colAlias
	})
}

func OnDuplicateKeyUpdate() *dupKeyUpdater {
	return &dupKeyUpdater{}
}

type dupKeyUpdater struct {
	sets []dialect.Set
}

func (s dupKeyUpdater) Apply(q *dialect.InsertQuery) {
	q.DuplicateKeyUpdate = append(q.DuplicateKeyUpdate, s.sets...)
}

func (s *dupKeyUpdater) SetCol(col string, val any) *dupKeyUpdater {
	s.sets = append(s.sets, dialect.Set{Col: col, Val: val})
	return s
}

func (s *dupKeyUpdater) Set(alias string, cols ...string) *dupKeyUpdater {
	newCols := make([]dialect.Set, len(cols))
	for i, c := range cols {
		newCols[i] = dialect.Set{Col: c, Val: expr.Quote(alias, c)}
	}

	s.sets = append(s.sets, newCols...)
	return s
}

func (s *dupKeyUpdater) SetValues(cols ...string) *dupKeyUpdater {
	newCols := make([]dialect.Set, len(cols))
	for i, c := range cols {
		newCols[i] = dialect.Set{Col: c, Val: dialect.NewFunction("VALUES", expr.Quote(c))}
	}

	s.sets = append(s.sets, newCols...)
	return s
}
