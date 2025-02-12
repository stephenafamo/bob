package dm

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.QBName[*dialect.DeleteQuery](name)
}

func SetVar(statement string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.SetVar[*dialect.DeleteQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*dialect.DeleteQuery] {
	return dialect.MaxExecutionTime[*dialect.DeleteQuery](n)
}

func ResourceGroup(name string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.ResourceGroup[*dialect.DeleteQuery](name)
}

func BKA(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.BKA[*dialect.DeleteQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoBKA[*dialect.DeleteQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.BNL[*dialect.DeleteQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoBNL[*dialect.DeleteQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.DerivedConditionPushdown[*dialect.DeleteQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoDerivedConditionPushdown[*dialect.DeleteQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.HashJoin[*dialect.DeleteQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoHashJoin[*dialect.DeleteQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.Merge[*dialect.DeleteQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoMerge[*dialect.DeleteQuery](tables...)
}

func Index(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.Index[*dialect.DeleteQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoIndex[*dialect.DeleteQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.GroupIndex[*dialect.DeleteQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoGroupIndex[*dialect.DeleteQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.JoinIndex[*dialect.DeleteQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoJoinIndex[*dialect.DeleteQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.OrderIndex[*dialect.DeleteQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoOrderIndex[*dialect.DeleteQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.IndexMerge[*dialect.DeleteQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoIndexMerge[*dialect.DeleteQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.MRR[*dialect.DeleteQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoMRR[*dialect.DeleteQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoICP[*dialect.DeleteQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoRangeOptimazation[*dialect.DeleteQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.SkipScan[*dialect.DeleteQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoSkipScan[*dialect.DeleteQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.Semijoin[*dialect.DeleteQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.NoSemijoin[*dialect.DeleteQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.Subquery[*dialect.DeleteQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.JoinFixedOrder[*dialect.DeleteQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.JoinOrder[*dialect.DeleteQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.JoinPrefix[*dialect.DeleteQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*dialect.DeleteQuery] {
	return dialect.JoinSuffix[*dialect.DeleteQuery](tables...)
}
