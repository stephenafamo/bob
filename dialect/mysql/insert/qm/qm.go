package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func Into(name any, columns ...string) bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.Table = name
		i.Columns = columns
	})
}

func LowPriority() bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func HighPriority() bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.AppendModifier("HIGH_PRIORITY")
	})
}

func Ignore() bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.AppendModifier("IGNORE")
	})
}

func Partition(partitions ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Partition[*mysql.InsertQuery](partitions...)
}

func Values(clauses ...any) bob.Mod[*mysql.InsertQuery] {
	return mods.Values[*mysql.InsertQuery](clauses)
}

// Insert from a query
func Query(q bob.Query) bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.Query = q
	})
}

// Insert with Set a = b
func Set(col string, val any) bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.Sets = append(i.Sets, mysql.Set{
			Col: col,
			Val: val,
		})
	})
}

func As(rowAlias string, colAlias ...string) bob.Mod[*mysql.InsertQuery] {
	return mods.QueryModFunc[*mysql.InsertQuery](func(i *mysql.InsertQuery) {
		i.RowAlias = rowAlias
		i.ColumnAlias = colAlias
	})
}

func OnDuplicateKeyUpdate() *dupKeyUpdater {
	return &dupKeyUpdater{}
}

type dupKeyUpdater struct {
	sets []mysql.Set
}

func (s dupKeyUpdater) Apply(q *mysql.InsertQuery) {
	q.DuplicateKeyUpdate = append(q.DuplicateKeyUpdate, s.sets...)
}

func (s *dupKeyUpdater) Set(col string, val any) *dupKeyUpdater {
	s.sets = append(s.sets, mysql.Set{Col: col, Val: val})
	return s
}
