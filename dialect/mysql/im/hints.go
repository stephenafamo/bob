package im

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*dialect.InsertQuery] {
	return dialect.QBName[*dialect.InsertQuery](name)
}

func SetVar(statement string) bob.Mod[*dialect.InsertQuery] {
	return dialect.SetVar[*dialect.InsertQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*dialect.InsertQuery] {
	return dialect.MaxExecutionTime[*dialect.InsertQuery](n)
}

func ResourceGroup(name string) bob.Mod[*dialect.InsertQuery] {
	return dialect.ResourceGroup[*dialect.InsertQuery](name)
}

func BKA(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.BKA[*dialect.InsertQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoBKA[*dialect.InsertQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.BNL[*dialect.InsertQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoBNL[*dialect.InsertQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.DerivedConditionPushdown[*dialect.InsertQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoDerivedConditionPushdown[*dialect.InsertQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.HashJoin[*dialect.InsertQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoHashJoin[*dialect.InsertQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.Merge[*dialect.InsertQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoMerge[*dialect.InsertQuery](tables...)
}

func Index(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.Index[*dialect.InsertQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoIndex[*dialect.InsertQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.GroupIndex[*dialect.InsertQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoGroupIndex[*dialect.InsertQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.JoinIndex[*dialect.InsertQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoJoinIndex[*dialect.InsertQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.OrderIndex[*dialect.InsertQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoOrderIndex[*dialect.InsertQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.IndexMerge[*dialect.InsertQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoIndexMerge[*dialect.InsertQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.MRR[*dialect.InsertQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoMRR[*dialect.InsertQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoICP[*dialect.InsertQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoRangeOptimazation[*dialect.InsertQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.SkipScan[*dialect.InsertQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoSkipScan[*dialect.InsertQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.Semijoin[*dialect.InsertQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.NoSemijoin[*dialect.InsertQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*dialect.InsertQuery] {
	return dialect.Subquery[*dialect.InsertQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*dialect.InsertQuery] {
	return dialect.JoinFixedOrder[*dialect.InsertQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.JoinOrder[*dialect.InsertQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.JoinPrefix[*dialect.InsertQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*dialect.InsertQuery] {
	return dialect.JoinSuffix[*dialect.InsertQuery](tables...)
}
