package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[dialect.SelectEditor] {
	return dialect.QBName[dialect.SelectEditor](name)
}

func SetVar(statement string) bob.Mod[dialect.SelectEditor] {
	return dialect.SetVar[dialect.SelectEditor](statement)
}

func MaxExecutionTime(n int) bob.Mod[dialect.SelectEditor] {
	return dialect.MaxExecutionTime[dialect.SelectEditor](n)
}

func ResourceGroup(name string) bob.Mod[dialect.SelectEditor] {
	return dialect.ResourceGroup[dialect.SelectEditor](name)
}

func BKA(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.BKA[dialect.SelectEditor](tables...)
}

func NoBKA(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoBKA[dialect.SelectEditor](tables...)
}

func BNL(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.BNL[dialect.SelectEditor](tables...)
}

func NoBNL(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoBNL[dialect.SelectEditor](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.DerivedConditionPushdown[dialect.SelectEditor](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoDerivedConditionPushdown[dialect.SelectEditor](tables...)
}

func HashJoin(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.HashJoin[dialect.SelectEditor](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoHashJoin[dialect.SelectEditor](tables...)
}

func Merge(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.Merge[dialect.SelectEditor](tables...)
}

func NoMerge(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoMerge[dialect.SelectEditor](tables...)
}

func Index(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.Index[dialect.SelectEditor](tables...)
}

func NoIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoIndex[dialect.SelectEditor](tables...)
}

func GroupIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.GroupIndex[dialect.SelectEditor](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoGroupIndex[dialect.SelectEditor](tables...)
}

func JoinIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.JoinIndex[dialect.SelectEditor](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoJoinIndex[dialect.SelectEditor](tables...)
}

func OrderIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.OrderIndex[dialect.SelectEditor](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoOrderIndex[dialect.SelectEditor](tables...)
}

func IndexMerge(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.IndexMerge[dialect.SelectEditor](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoIndexMerge[dialect.SelectEditor](tables...)
}

func MRR(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.MRR[dialect.SelectEditor](tables...)
}

func NoMRR(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoMRR[dialect.SelectEditor](tables...)
}

func NoICP(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoICP[dialect.SelectEditor](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoRangeOptimazation[dialect.SelectEditor](tables...)
}

func SkipScan(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.SkipScan[dialect.SelectEditor](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoSkipScan[dialect.SelectEditor](tables...)
}

func Semijoin(strategy ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.Semijoin[dialect.SelectEditor](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.NoSemijoin[dialect.SelectEditor](strategy...)
}

func Subquery(strategy string) bob.Mod[dialect.SelectEditor] {
	return dialect.Subquery[dialect.SelectEditor](strategy)
}

func JoinFixedOrder(name string) bob.Mod[dialect.SelectEditor] {
	return dialect.JoinFixedOrder[dialect.SelectEditor](name)
}

func JoinOrder(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.JoinOrder[dialect.SelectEditor](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.JoinPrefix[dialect.SelectEditor](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[dialect.SelectEditor] {
	return dialect.JoinSuffix[dialect.SelectEditor](tables...)
}
