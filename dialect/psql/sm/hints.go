package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func SeqScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.SeqScan[*dialect.SelectQuery](table)
}

func NoSeqScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoSeqScan[*dialect.SelectQuery](table)
}

func IndexScan(table string, indexes ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.IndexScan[*dialect.SelectQuery](table, indexes...)
}

func NoIndexScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoIndexScan[*dialect.SelectQuery](table)
}

func IndexOnlyScan(table string, indexes ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.IndexOnlyScan[*dialect.SelectQuery](table, indexes...)
}

func NoIndexOnlyScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoIndexOnlyScan[*dialect.SelectQuery](table)
}

func BitmapScan(table string, indexes ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.BitmapScan[*dialect.SelectQuery](table, indexes...)
}

func NoBitmapScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoBitmapScan[*dialect.SelectQuery](table)
}

func TidScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.TidScan[*dialect.SelectQuery](table)
}

func NoTidScan(table string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoTidScan[*dialect.SelectQuery](table)
}

func NestLoop(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NestLoop[*dialect.SelectQuery](tables...)
}

func NoNestLoop(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoNestLoop[*dialect.SelectQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.HashJoin[*dialect.SelectQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoHashJoin[*dialect.SelectQuery](tables...)
}

func MergeJoin(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.MergeJoin[*dialect.SelectQuery](tables...)
}

func NoMergeJoin(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoMergeJoin[*dialect.SelectQuery](tables...)
}

func Leading(spec string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Leading[*dialect.SelectQuery](spec)
}

func Rows(spec string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Rows[*dialect.SelectQuery](spec)
}

func Parallel(table string, nworkers int, strength string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Parallel[*dialect.SelectQuery](table, nworkers, strength)
}

func Set(variable string, value string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Set[*dialect.SelectQuery](variable, value)
}
