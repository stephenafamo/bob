package mysql

import (
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

type hints struct {
	hints []string
}

func (h *hints) AppendHint(hint string) {
	h.hints = append(h.hints, hint)
}

func (h hints) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, h.hints, "/*+ ", "\n    ", " */")
}

type hintMod[Q interface{ AppendHint(string) }] struct{}

func (hintMod[Q]) QBName(name string) query.Mod[Q] {
	hint := fmt.Sprintf("QB_NAME(%s)", name)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) SetVar(statement string) query.Mod[Q] {
	hint := fmt.Sprintf("SET_VAR(%s)", statement)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) MaxExecutionTime(n int) query.Mod[Q] {
	hint := fmt.Sprintf("MAX_EXECUTION_TIME(%d)", n)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) ResourceGroup(name string) query.Mod[Q] {
	hint := fmt.Sprintf("RESOURCE_GROUP(%s)", name)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) BKA(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("BKA(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoBKA(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_BKA(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) BNL(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("BNL(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoBNL(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_BNL(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) DerivedConditionPushdown(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("DERIVED_CONDITION_PUSHDOWN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoDerivedConditionPushdown(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_DERIVED_CONDITION_PUSHDOWN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) HashJoin(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("HASH_JOIN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoHashJoin(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_HASH_JOIN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Merge(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoMerge(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Index(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) GroupIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("GROUP_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoGroupIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_GROUP_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("JOIN_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoJoinIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_JOIN_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) OrderIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("ORDER_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoOrderIndex(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_ORDER_INDEX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) IndexMerge(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("INDEX_MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoIndexMerge(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_INDEX_MERGE(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) MRR(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("MRR(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoMRR(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_MRR(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoICP(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_ICP(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoRangeOptimazation(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_RANGE_OPTIMAZATION(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) SkipScan(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("SKIP_SCAN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoSkipScan(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_SKIP_SCAN(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Semijoin(strategy ...string) query.Mod[Q] {
	hint := fmt.Sprintf("SEMIJOIN(%s)", strings.Join(strategy, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) NoSemijoin(strategy ...string) query.Mod[Q] {
	hint := fmt.Sprintf("NO_SEMIJOIN(%s)", strings.Join(strategy, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) Subquery(strategy string) query.Mod[Q] {
	hint := fmt.Sprintf("SUBQUERY(%s)", strategy)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinFixedOrder(name string) query.Mod[Q] {
	hint := fmt.Sprintf("JOIN_FIXED_ORDER(%s)", name)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinOrder(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("JOIN_ORDER(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinPrefix(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("JOIN_PREFIX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) JoinSuffix(tables ...string) query.Mod[Q] {
	hint := fmt.Sprintf("JOIN_SUFFIX(%s)", strings.Join(tables, ", "))
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}
