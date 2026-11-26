package dm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[dialect.DeleteEditor] {
	return dialect.QBName[dialect.DeleteEditor](name)
}

func SetVar(statement string) bob.Mod[dialect.DeleteEditor] {
	return dialect.SetVar[dialect.DeleteEditor](statement)
}

func MaxExecutionTime(n int) bob.Mod[dialect.DeleteEditor] {
	return dialect.MaxExecutionTime[dialect.DeleteEditor](n)
}

func ResourceGroup(name string) bob.Mod[dialect.DeleteEditor] {
	return dialect.ResourceGroup[dialect.DeleteEditor](name)
}

func BKA(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.BKA[dialect.DeleteEditor](tables...)
}

func NoBKA(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoBKA[dialect.DeleteEditor](tables...)
}

func BNL(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.BNL[dialect.DeleteEditor](tables...)
}

func NoBNL(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoBNL[dialect.DeleteEditor](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.DerivedConditionPushdown[dialect.DeleteEditor](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoDerivedConditionPushdown[dialect.DeleteEditor](tables...)
}

func HashJoin(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.HashJoin[dialect.DeleteEditor](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoHashJoin[dialect.DeleteEditor](tables...)
}

func Merge(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.Merge[dialect.DeleteEditor](tables...)
}

func NoMerge(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoMerge[dialect.DeleteEditor](tables...)
}

func Index(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.Index[dialect.DeleteEditor](tables...)
}

func NoIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoIndex[dialect.DeleteEditor](tables...)
}

func GroupIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.GroupIndex[dialect.DeleteEditor](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoGroupIndex[dialect.DeleteEditor](tables...)
}

func JoinIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.JoinIndex[dialect.DeleteEditor](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoJoinIndex[dialect.DeleteEditor](tables...)
}

func OrderIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.OrderIndex[dialect.DeleteEditor](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoOrderIndex[dialect.DeleteEditor](tables...)
}

func IndexMerge(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.IndexMerge[dialect.DeleteEditor](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoIndexMerge[dialect.DeleteEditor](tables...)
}

func MRR(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.MRR[dialect.DeleteEditor](tables...)
}

func NoMRR(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoMRR[dialect.DeleteEditor](tables...)
}

func NoICP(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoICP[dialect.DeleteEditor](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoRangeOptimazation[dialect.DeleteEditor](tables...)
}

func SkipScan(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.SkipScan[dialect.DeleteEditor](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoSkipScan[dialect.DeleteEditor](tables...)
}

func Semijoin(strategy ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.Semijoin[dialect.DeleteEditor](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.NoSemijoin[dialect.DeleteEditor](strategy...)
}

func Subquery(strategy string) bob.Mod[dialect.DeleteEditor] {
	return dialect.Subquery[dialect.DeleteEditor](strategy)
}

func JoinFixedOrder(name string) bob.Mod[dialect.DeleteEditor] {
	return dialect.JoinFixedOrder[dialect.DeleteEditor](name)
}

func JoinOrder(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.JoinOrder[dialect.DeleteEditor](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.JoinPrefix[dialect.DeleteEditor](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[dialect.DeleteEditor] {
	return dialect.JoinSuffix[dialect.DeleteEditor](tables...)
}
