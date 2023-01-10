package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*dialect.SelectQuery] {
	return dialect.QBName[*dialect.SelectQuery](name)
}

func SetVar(statement string) bob.Mod[*dialect.SelectQuery] {
	return dialect.SetVar[*dialect.SelectQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*dialect.SelectQuery] {
	return dialect.MaxExecutionTime[*dialect.SelectQuery](n)
}

func ResourceGroup(name string) bob.Mod[*dialect.SelectQuery] {
	return dialect.ResourceGroup[*dialect.SelectQuery](name)
}

func BKA(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.BKA[*dialect.SelectQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoBKA[*dialect.SelectQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.BNL[*dialect.SelectQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoBNL[*dialect.SelectQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.DerivedConditionPushdown[*dialect.SelectQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoDerivedConditionPushdown[*dialect.SelectQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.HashJoin[*dialect.SelectQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoHashJoin[*dialect.SelectQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Merge[*dialect.SelectQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoMerge[*dialect.SelectQuery](tables...)
}

func Index(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Index[*dialect.SelectQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoIndex[*dialect.SelectQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.GroupIndex[*dialect.SelectQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoGroupIndex[*dialect.SelectQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.JoinIndex[*dialect.SelectQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoJoinIndex[*dialect.SelectQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.OrderIndex[*dialect.SelectQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoOrderIndex[*dialect.SelectQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.IndexMerge[*dialect.SelectQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoIndexMerge[*dialect.SelectQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.MRR[*dialect.SelectQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoMRR[*dialect.SelectQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoICP[*dialect.SelectQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoRangeOptimazation[*dialect.SelectQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.SkipScan[*dialect.SelectQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoSkipScan[*dialect.SelectQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Semijoin[*dialect.SelectQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.NoSemijoin[*dialect.SelectQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*dialect.SelectQuery] {
	return dialect.Subquery[*dialect.SelectQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*dialect.SelectQuery] {
	return dialect.JoinFixedOrder[*dialect.SelectQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.JoinOrder[*dialect.SelectQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.JoinPrefix[*dialect.SelectQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*dialect.SelectQuery] {
	return dialect.JoinSuffix[*dialect.SelectQuery](tables...)
}
