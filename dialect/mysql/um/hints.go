package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.QBName[*dialect.UpdateQuery](name)
}

func SetVar(statement string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.SetVar[*dialect.UpdateQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*dialect.UpdateQuery] {
	return dialect.MaxExecutionTime[*dialect.UpdateQuery](n)
}

func ResourceGroup(name string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.ResourceGroup[*dialect.UpdateQuery](name)
}

func BKA(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.BKA[*dialect.UpdateQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoBKA[*dialect.UpdateQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.BNL[*dialect.UpdateQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoBNL[*dialect.UpdateQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.DerivedConditionPushdown[*dialect.UpdateQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoDerivedConditionPushdown[*dialect.UpdateQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.HashJoin[*dialect.UpdateQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoHashJoin[*dialect.UpdateQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.Merge[*dialect.UpdateQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoMerge[*dialect.UpdateQuery](tables...)
}

func Index(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.Index[*dialect.UpdateQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoIndex[*dialect.UpdateQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.GroupIndex[*dialect.UpdateQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoGroupIndex[*dialect.UpdateQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.JoinIndex[*dialect.UpdateQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoJoinIndex[*dialect.UpdateQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.OrderIndex[*dialect.UpdateQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoOrderIndex[*dialect.UpdateQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.IndexMerge[*dialect.UpdateQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoIndexMerge[*dialect.UpdateQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.MRR[*dialect.UpdateQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoMRR[*dialect.UpdateQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoICP[*dialect.UpdateQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoRangeOptimazation[*dialect.UpdateQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.SkipScan[*dialect.UpdateQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoSkipScan[*dialect.UpdateQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.Semijoin[*dialect.UpdateQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.NoSemijoin[*dialect.UpdateQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.Subquery[*dialect.UpdateQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.JoinFixedOrder[*dialect.UpdateQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.JoinOrder[*dialect.UpdateQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.JoinPrefix[*dialect.UpdateQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*dialect.UpdateQuery] {
	return dialect.JoinSuffix[*dialect.UpdateQuery](tables...)
}
