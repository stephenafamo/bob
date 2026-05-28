package dialect

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob"
)

type hints struct {
	hints []string
}

func (h *hints) AppendHint(hint string) {
	h.hints = append(h.hints, hint)
}

func (h hints) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, h.hints, "/*+ ", "\n    ", " */")
}

type hintable interface{ AppendHint(string) }

// hintQuotedTables formats table identifiers for optimizer hints using dialect quoting.
func hintQuotedTables(tables ...string) string {
	var b strings.Builder
	for i, t := range tables {
		if i != 0 {
			b.WriteString(", ")
		}
		Dialect.WriteQuoted(&b, t)
	}
	return b.String()
}

func QBName[Q hintable](name string) bob.Mod[Q] {
	hint := fmt.Sprintf("QB_NAME(%s)", name)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func SetVar[Q hintable](statement string) bob.Mod[Q] {
	hint := fmt.Sprintf("SET_VAR(%s)", statement)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func MaxExecutionTime[Q hintable](n int) bob.Mod[Q] {
	hint := fmt.Sprintf("MAX_EXECUTION_TIME(%d)", n)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func ResourceGroup[Q hintable](name string) bob.Mod[Q] {
	hint := fmt.Sprintf("RESOURCE_GROUP(%s)", name)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func BKA[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("BKA(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoBKA[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_BKA(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func BNL[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("BNL(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoBNL[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_BNL(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func DerivedConditionPushdown[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("DERIVED_CONDITION_PUSHDOWN(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoDerivedConditionPushdown[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_DERIVED_CONDITION_PUSHDOWN(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func HashJoin[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("HASH_JOIN(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoHashJoin[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_HASH_JOIN(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func Merge[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("MERGE(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoMerge[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_MERGE(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func Index[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func GroupIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("GROUP_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoGroupIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_GROUP_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func JoinIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoJoinIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_JOIN_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func OrderIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("ORDER_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoOrderIndex[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_ORDER_INDEX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func IndexMerge[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("INDEX_MERGE(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoIndexMerge[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_INDEX_MERGE(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func MRR[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("MRR(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoMRR[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_MRR(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoICP[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_ICP(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoRangeOptimazation[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_RANGE_OPTIMAZATION(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func SkipScan[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("SKIP_SCAN(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoSkipScan[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_SKIP_SCAN(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func Semijoin[Q hintable](strategy ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("SEMIJOIN(%s)", strings.Join(strategy, ", "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoSemijoin[Q hintable](strategy ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_SEMIJOIN(%s)", strings.Join(strategy, ", "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func Subquery[Q hintable](strategy string) bob.Mod[Q] {
	hint := fmt.Sprintf("SUBQUERY(%s)", strategy)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func JoinFixedOrder[Q hintable](name string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_FIXED_ORDER(%s)", name)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func JoinOrder[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_ORDER(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func JoinPrefix[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_PREFIX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func JoinSuffix[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_SUFFIX(%s)", hintQuotedTables(tables...))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}
