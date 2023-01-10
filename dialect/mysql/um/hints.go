package um

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func QBName(name string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.QBName[*mysql.UpdateQuery](name)
}

func SetVar(statement string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.SetVar[*mysql.UpdateQuery](statement)
}

func MaxExecutionTime(n int) bob.Mod[*mysql.UpdateQuery] {
	return dialect.MaxExecutionTime[*mysql.UpdateQuery](n)
}

func ResourceGroup(name string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.ResourceGroup[*mysql.UpdateQuery](name)
}

func BKA(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.BKA[*mysql.UpdateQuery](tables...)
}

func NoBKA(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoBKA[*mysql.UpdateQuery](tables...)
}

func BNL(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.BNL[*mysql.UpdateQuery](tables...)
}

func NoBNL(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoBNL[*mysql.UpdateQuery](tables...)
}

func DerivedConditionPushdown(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.DerivedConditionPushdown[*mysql.UpdateQuery](tables...)
}

func NoDerivedConditionPushdown(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoDerivedConditionPushdown[*mysql.UpdateQuery](tables...)
}

func HashJoin(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.HashJoin[*mysql.UpdateQuery](tables...)
}

func NoHashJoin(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoHashJoin[*mysql.UpdateQuery](tables...)
}

func Merge(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.Merge[*mysql.UpdateQuery](tables...)
}

func NoMerge(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoMerge[*mysql.UpdateQuery](tables...)
}

func Index(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.Index[*mysql.UpdateQuery](tables...)
}

func NoIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoIndex[*mysql.UpdateQuery](tables...)
}

func GroupIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.GroupIndex[*mysql.UpdateQuery](tables...)
}

func NoGroupIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoGroupIndex[*mysql.UpdateQuery](tables...)
}

func JoinIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.JoinIndex[*mysql.UpdateQuery](tables...)
}

func NoJoinIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoJoinIndex[*mysql.UpdateQuery](tables...)
}

func OrderIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.OrderIndex[*mysql.UpdateQuery](tables...)
}

func NoOrderIndex(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoOrderIndex[*mysql.UpdateQuery](tables...)
}

func IndexMerge(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.IndexMerge[*mysql.UpdateQuery](tables...)
}

func NoIndexMerge(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoIndexMerge[*mysql.UpdateQuery](tables...)
}

func MRR(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.MRR[*mysql.UpdateQuery](tables...)
}

func NoMRR(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoMRR[*mysql.UpdateQuery](tables...)
}

func NoICP(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoICP[*mysql.UpdateQuery](tables...)
}

func NoRangeOptimazation(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoRangeOptimazation[*mysql.UpdateQuery](tables...)
}

func SkipScan(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.SkipScan[*mysql.UpdateQuery](tables...)
}

func NoSkipScan(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoSkipScan[*mysql.UpdateQuery](tables...)
}

func Semijoin(strategy ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.Semijoin[*mysql.UpdateQuery](strategy...)
}

func NoSemijoin(strategy ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.NoSemijoin[*mysql.UpdateQuery](strategy...)
}

func Subquery(strategy string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.Subquery[*mysql.UpdateQuery](strategy)
}

func JoinFixedOrder(name string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.JoinFixedOrder[*mysql.UpdateQuery](name)
}

func JoinOrder(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.JoinOrder[*mysql.UpdateQuery](tables...)
}

func JoinPrefix(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.JoinPrefix[*mysql.UpdateQuery](tables...)
}

func JoinSuffix(tables ...string) bob.Mod[*mysql.UpdateQuery] {
	return dialect.JoinSuffix[*mysql.UpdateQuery](tables...)
}
