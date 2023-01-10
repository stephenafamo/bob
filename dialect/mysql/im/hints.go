package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*mysql.InsertQuery] {
	return dialect.QBName[*mysql.InsertQuery](name)
}

func SetVar(statement string) bob.Mod[*mysql.InsertQuery] {
	return dialect.SetVar[*mysql.InsertQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*mysql.InsertQuery] {
	return dialect.MaxExecutionTime[*mysql.InsertQuery](n)
}

func ResourceGroup(name string) bob.Mod[*mysql.InsertQuery] {
	return dialect.ResourceGroup[*mysql.InsertQuery](name)
}

func BKA(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.BKA[*mysql.InsertQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoBKA[*mysql.InsertQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.BNL[*mysql.InsertQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoBNL[*mysql.InsertQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.DerivedConditionPushdown[*mysql.InsertQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoDerivedConditionPushdown[*mysql.InsertQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.HashJoin[*mysql.InsertQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoHashJoin[*mysql.InsertQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Merge[*mysql.InsertQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoMerge[*mysql.InsertQuery](tables...)
}

func Index(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Index[*mysql.InsertQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoIndex[*mysql.InsertQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.GroupIndex[*mysql.InsertQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoGroupIndex[*mysql.InsertQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.JoinIndex[*mysql.InsertQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoJoinIndex[*mysql.InsertQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.OrderIndex[*mysql.InsertQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoOrderIndex[*mysql.InsertQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.IndexMerge[*mysql.InsertQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoIndexMerge[*mysql.InsertQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.MRR[*mysql.InsertQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoMRR[*mysql.InsertQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoICP[*mysql.InsertQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoRangeOptimazation[*mysql.InsertQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.SkipScan[*mysql.InsertQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoSkipScan[*mysql.InsertQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Semijoin[*mysql.InsertQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.NoSemijoin[*mysql.InsertQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*mysql.InsertQuery] {
	return dialect.Subquery[*mysql.InsertQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*mysql.InsertQuery] {
	return dialect.JoinFixedOrder[*mysql.InsertQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.JoinOrder[*mysql.InsertQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.JoinPrefix[*mysql.InsertQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*mysql.InsertQuery] {
	return dialect.JoinSuffix[*mysql.InsertQuery](tables...)
}
