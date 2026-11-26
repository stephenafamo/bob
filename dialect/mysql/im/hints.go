package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[dialect.InsertEditor] {
	return dialect.QBName[dialect.InsertEditor](name)
}

func SetVar(statement string) bob.Mod[dialect.InsertEditor] {
	return dialect.SetVar[dialect.InsertEditor](statement)
}

func MaxExecutionTime(n int) bob.Mod[dialect.InsertEditor] {
	return dialect.MaxExecutionTime[dialect.InsertEditor](n)
}

func ResourceGroup(name string) bob.Mod[dialect.InsertEditor] {
	return dialect.ResourceGroup[dialect.InsertEditor](name)
}

func BKA(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.BKA[dialect.InsertEditor](tables...)
}

func NoBKA(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoBKA[dialect.InsertEditor](tables...)
}

func BNL(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.BNL[dialect.InsertEditor](tables...)
}

func NoBNL(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoBNL[dialect.InsertEditor](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.DerivedConditionPushdown[dialect.InsertEditor](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoDerivedConditionPushdown[dialect.InsertEditor](tables...)
}

func HashJoin(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.HashJoin[dialect.InsertEditor](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoHashJoin[dialect.InsertEditor](tables...)
}

func Merge(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.Merge[dialect.InsertEditor](tables...)
}

func NoMerge(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoMerge[dialect.InsertEditor](tables...)
}

func Index(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.Index[dialect.InsertEditor](tables...)
}

func NoIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoIndex[dialect.InsertEditor](tables...)
}

func GroupIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.GroupIndex[dialect.InsertEditor](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoGroupIndex[dialect.InsertEditor](tables...)
}

func JoinIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.JoinIndex[dialect.InsertEditor](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoJoinIndex[dialect.InsertEditor](tables...)
}

func OrderIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.OrderIndex[dialect.InsertEditor](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoOrderIndex[dialect.InsertEditor](tables...)
}

func IndexMerge(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.IndexMerge[dialect.InsertEditor](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoIndexMerge[dialect.InsertEditor](tables...)
}

func MRR(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.MRR[dialect.InsertEditor](tables...)
}

func NoMRR(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoMRR[dialect.InsertEditor](tables...)
}

func NoICP(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoICP[dialect.InsertEditor](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoRangeOptimazation[dialect.InsertEditor](tables...)
}

func SkipScan(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.SkipScan[dialect.InsertEditor](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoSkipScan[dialect.InsertEditor](tables...)
}

func Semijoin(strategy ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.Semijoin[dialect.InsertEditor](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.NoSemijoin[dialect.InsertEditor](strategy...)
}

func Subquery(strategy string) bob.Mod[dialect.InsertEditor] {
	return dialect.Subquery[dialect.InsertEditor](strategy)
}

func JoinFixedOrder(name string) bob.Mod[dialect.InsertEditor] {
	return dialect.JoinFixedOrder[dialect.InsertEditor](name)
}

func JoinOrder(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.JoinOrder[dialect.InsertEditor](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.JoinPrefix[dialect.InsertEditor](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[dialect.InsertEditor] {
	return dialect.JoinSuffix[dialect.InsertEditor](tables...)
}
