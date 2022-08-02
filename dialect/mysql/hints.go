package mysql

import (
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/mods"
)

type hints struct {
	hints []string
}

func (h *hints) AppendHint(hint string) {
	h.hints = append(h.hints, hint)
}

func (h hints) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, h.hints, "/*+ ", "\n    ", " */")
}

type hintMod[Q interface{ AppendHint(string) }] struct{}

func (hintMod[Q]) QBName(name string) bob.Mod[Q] {
	hint := fmt.Sprintf("QB_NAME(%s)", name)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) SetVar(statement string) bob.Mod[Q] {
	hint := fmt.Sprintf("SET_VAR(%s)", statement)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) MaxExecutionTime(n int) bob.Mod[Q] {
	hint := fmt.Sprintf("MAX_EXECUTION_TIME(%d)", n)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) ResourceGroup(name string) bob.Mod[Q] {
	hint := fmt.Sprintf("RESOURCE_GROUP(%s)", name)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) BKA(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("BKA(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoBKA(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_BKA(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) BNL(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("BNL(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoBNL(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_BNL(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) DerivedConditionPushdown(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("DERIVED_CONDITION_PUSHDOWN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoDerivedConditionPushdown(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_DERIVED_CONDITION_PUSHDOWN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) HashJoin(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("HASH_JOIN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoHashJoin(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_HASH_JOIN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Merge(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoMerge(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Index(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) GroupIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("GROUP_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoGroupIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_GROUP_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoJoinIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_JOIN_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) OrderIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("ORDER_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoOrderIndex(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_ORDER_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) IndexMerge(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("INDEX_MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoIndexMerge(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_INDEX_MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) MRR(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("MRR(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoMRR(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_MRR(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoICP(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_ICP(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoRangeOptimazation(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_RANGE_OPTIMAZATION(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) SkipScan(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("SKIP_SCAN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoSkipScan(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_SKIP_SCAN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Semijoin(strategy ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("SEMIJOIN(%s)", strings.Join(strategy, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoSemijoin(strategy ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NO_SEMIJOIN(%s)", strings.Join(strategy, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Subquery(strategy string) bob.Mod[Q] {
	hint := fmt.Sprintf("SUBQUERY(%s)", strategy)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinFixedOrder(name string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_FIXED_ORDER(%s)", name)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinOrder(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_ORDER(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinPrefix(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_PREFIX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinSuffix(tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("JOIN_SUFFIX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}
