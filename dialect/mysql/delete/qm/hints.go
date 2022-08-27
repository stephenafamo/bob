package qm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.QBName[*mysql.DeleteQuery](name)
}

func SetVar(statement string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.SetVar[*mysql.DeleteQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*mysql.DeleteQuery] {
	return dialect.MaxExecutionTime[*mysql.DeleteQuery](n)
}

func ResourceGroup(name string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.ResourceGroup[*mysql.DeleteQuery](name)
}

func BKA(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.BKA[*mysql.DeleteQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoBKA[*mysql.DeleteQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.BNL[*mysql.DeleteQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoBNL[*mysql.DeleteQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.DerivedConditionPushdown[*mysql.DeleteQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoDerivedConditionPushdown[*mysql.DeleteQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.HashJoin[*mysql.DeleteQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoHashJoin[*mysql.DeleteQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.Merge[*mysql.DeleteQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoMerge[*mysql.DeleteQuery](tables...)
}

func Index(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.Index[*mysql.DeleteQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoIndex[*mysql.DeleteQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.GroupIndex[*mysql.DeleteQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoGroupIndex[*mysql.DeleteQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.JoinIndex[*mysql.DeleteQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoJoinIndex[*mysql.DeleteQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.OrderIndex[*mysql.DeleteQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoOrderIndex[*mysql.DeleteQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.IndexMerge[*mysql.DeleteQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoIndexMerge[*mysql.DeleteQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.MRR[*mysql.DeleteQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoMRR[*mysql.DeleteQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoICP[*mysql.DeleteQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoRangeOptimazation[*mysql.DeleteQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.SkipScan[*mysql.DeleteQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoSkipScan[*mysql.DeleteQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.Semijoin[*mysql.DeleteQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.NoSemijoin[*mysql.DeleteQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.Subquery[*mysql.DeleteQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.JoinFixedOrder[*mysql.DeleteQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.JoinOrder[*mysql.DeleteQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.JoinPrefix[*mysql.DeleteQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*mysql.DeleteQuery] {
	return dialect.JoinSuffix[*mysql.DeleteQuery](tables...)
}
