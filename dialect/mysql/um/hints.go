package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[dialect.UpdateEditor] {
	return dialect.QBName[dialect.UpdateEditor](name)
}

func SetVar(statement string) bob.Mod[dialect.UpdateEditor] {
	return dialect.SetVar[dialect.UpdateEditor](statement)
}

func MaxExecutionTime(n int) bob.Mod[dialect.UpdateEditor] {
	return dialect.MaxExecutionTime[dialect.UpdateEditor](n)
}

func ResourceGroup(name string) bob.Mod[dialect.UpdateEditor] {
	return dialect.ResourceGroup[dialect.UpdateEditor](name)
}

func BKA(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.BKA[dialect.UpdateEditor](tables...)
}

func NoBKA(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoBKA[dialect.UpdateEditor](tables...)
}

func BNL(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.BNL[dialect.UpdateEditor](tables...)
}

func NoBNL(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoBNL[dialect.UpdateEditor](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.DerivedConditionPushdown[dialect.UpdateEditor](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoDerivedConditionPushdown[dialect.UpdateEditor](tables...)
}

func HashJoin(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.HashJoin[dialect.UpdateEditor](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoHashJoin[dialect.UpdateEditor](tables...)
}

func Merge(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.Merge[dialect.UpdateEditor](tables...)
}

func NoMerge(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoMerge[dialect.UpdateEditor](tables...)
}

func Index(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.Index[dialect.UpdateEditor](tables...)
}

func NoIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoIndex[dialect.UpdateEditor](tables...)
}

func GroupIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.GroupIndex[dialect.UpdateEditor](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoGroupIndex[dialect.UpdateEditor](tables...)
}

func JoinIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.JoinIndex[dialect.UpdateEditor](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoJoinIndex[dialect.UpdateEditor](tables...)
}

func OrderIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.OrderIndex[dialect.UpdateEditor](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoOrderIndex[dialect.UpdateEditor](tables...)
}

func IndexMerge(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.IndexMerge[dialect.UpdateEditor](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoIndexMerge[dialect.UpdateEditor](tables...)
}

func MRR(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.MRR[dialect.UpdateEditor](tables...)
}

func NoMRR(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoMRR[dialect.UpdateEditor](tables...)
}

func NoICP(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoICP[dialect.UpdateEditor](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoRangeOptimazation[dialect.UpdateEditor](tables...)
}

func SkipScan(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.SkipScan[dialect.UpdateEditor](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoSkipScan[dialect.UpdateEditor](tables...)
}

func Semijoin(strategy ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.Semijoin[dialect.UpdateEditor](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.NoSemijoin[dialect.UpdateEditor](strategy...)
}

func Subquery(strategy string) bob.Mod[dialect.UpdateEditor] {
	return dialect.Subquery[dialect.UpdateEditor](strategy)
}

func JoinFixedOrder(name string) bob.Mod[dialect.UpdateEditor] {
	return dialect.JoinFixedOrder[dialect.UpdateEditor](name)
}

func JoinOrder(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.JoinOrder[dialect.UpdateEditor](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.JoinPrefix[dialect.UpdateEditor](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[dialect.UpdateEditor] {
	return dialect.JoinSuffix[dialect.UpdateEditor](tables...)
}
