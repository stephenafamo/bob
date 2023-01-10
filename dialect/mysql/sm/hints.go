package sm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*mysql.SelectQuery] {
	return dialect.QBName[*mysql.SelectQuery](name)
}

func SetVar(statement string) bob.Mod[*mysql.SelectQuery] {
	return dialect.SetVar[*mysql.SelectQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*mysql.SelectQuery] {
	return dialect.MaxExecutionTime[*mysql.SelectQuery](n)
}

func ResourceGroup(name string) bob.Mod[*mysql.SelectQuery] {
	return dialect.ResourceGroup[*mysql.SelectQuery](name)
}

func BKA(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.BKA[*mysql.SelectQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoBKA[*mysql.SelectQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.BNL[*mysql.SelectQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoBNL[*mysql.SelectQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.DerivedConditionPushdown[*mysql.SelectQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoDerivedConditionPushdown[*mysql.SelectQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.HashJoin[*mysql.SelectQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoHashJoin[*mysql.SelectQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.Merge[*mysql.SelectQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoMerge[*mysql.SelectQuery](tables...)
}

func Index(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.Index[*mysql.SelectQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoIndex[*mysql.SelectQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.GroupIndex[*mysql.SelectQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoGroupIndex[*mysql.SelectQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.JoinIndex[*mysql.SelectQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoJoinIndex[*mysql.SelectQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.OrderIndex[*mysql.SelectQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoOrderIndex[*mysql.SelectQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.IndexMerge[*mysql.SelectQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoIndexMerge[*mysql.SelectQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.MRR[*mysql.SelectQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoMRR[*mysql.SelectQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoICP[*mysql.SelectQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoRangeOptimazation[*mysql.SelectQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.SkipScan[*mysql.SelectQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoSkipScan[*mysql.SelectQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.Semijoin[*mysql.SelectQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.NoSemijoin[*mysql.SelectQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*mysql.SelectQuery] {
	return dialect.Subquery[*mysql.SelectQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*mysql.SelectQuery] {
	return dialect.JoinFixedOrder[*mysql.SelectQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.JoinOrder[*mysql.SelectQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.JoinPrefix[*mysql.SelectQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*mysql.SelectQuery] {
	return dialect.JoinSuffix[*mysql.SelectQuery](tables...)
}
