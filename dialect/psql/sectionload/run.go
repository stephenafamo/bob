package sectionload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/fm"
	"github.com/stephenafamo/bob/expr"
)

// checkMapperMods rejects a parent query carrying one or more
// bob.MapperMod values (issue P-06): sectionload's own scan never
// consumes them, so left unchecked they succeed silently with the
// relation simply missing from the result, rather than erroring. Mod
// identity is never inspected (no reflection-based classification of
// specific mods, issue P-06 §9 step 3) — any non-zero count is rejected,
// uniformly, regardless of what added it.
//
// P-13では、親SELECTから読み取ったcount宣言がある場合に限り、
// MapperMod数が宣言数と一致すれば許可する。sectionloadはMapperMod自体を
// 実行せず、count列を自前で走査して生成モデルのPreloadCountへ渡す。
//
// 現行bob APIではMapperModの由来を識別できないため、これは件数による
// 代理判定であって由来の証明ではない。SetMapperModsで別種のMapperModへ
// 同数差し替えされた場合は検出できない。この残存リスクを閉じるには、
// bob側でMapperModの種別または宣言と共有する識別子を公開する必要がある
// (Codexレビュー1回目・指摘2、2026-08-20。詳細は設計書§6.5参照)。
//
// totalPreloadColumns(Codexレビュー1回目・防御強化案(a)、2026-08-20)は
// 追加の防御条件: 宣言数とMapperMod数の一致に加えて、親クエリの
// PreloadColumns自体の総数も宣言数と一致することを求める。これにより、
// 「宣言数とMapperMod数はたまたま一致するが、PreloadColumns自体に宣言外
// の列(利用者が独自に足した別のPreloadCount以外のJOIN Preload等)が
// 混ざっている」ケースを保守的に拒否できる。副作用として、count以外の
// PreloadColumnsを持つ親クエリ(通常の利用では起こらない組み合わせ)は
// 保守的に拒否される。ただしこれもMapperMod自体の由来を型的に証明する
// ものではなく、あくまで追加の代理条件であることに変わりはない
// (§6.5の残存リスク参照)。
func checkMapperMods(parentQuery bob.Query, declared []declaredCountPreload, totalPreloadColumns int) error {
	modder, ok := parentQuery.(bob.MapperModder)
	if !ok {
		return nil
	}
	n := len(modder.GetMapperMods())
	if n == 0 {
		return nil
	}
	if n == len(declared) && totalPreloadColumns == len(declared) {
		// P-13: 由来は検証不能なため、宣言数・PreloadColumns総数との一致を
		// 代理条件として許可する。同数差し替えの残存リスクは関数コメント
		// のとおり。
		return nil
	}
	return &UnsupportedMapperModsError{Count: n}
}

// psqlSelectModdable is satisfied by *dialect.SelectQuery (and anything
// bob generates on top of it, e.g. models.XQuery types), whose Apply is a
// non-generic method — the type parameter on bob.Mod[*dialect.SelectQuery]
// resolves at the interface boundary, not inside this function, so no type
// parameter is needed here.
type psqlSelectModdable interface {
	Apply(...bob.Mod[*dialect.SelectQuery])
}

// installBuildTimeMapperModCheck wraps every contextual mod already on the
// parent query so that GetMapperMods() is checked immediately after each
// one runs, and returns a function that restores the original mod slice.
// SelectQuery.WriteSQL runs contextual mods in order on its build copy, so
// this observes MapperMods added by a real Preloader.Apply run through
// AppendContextualModFunc — the gap checkMapperMods cannot see, because it
// runs before the first bob.Build (P-06 round 2).
//
// Checking after EACH mod, not once at the end, is deliberate: a later
// contextual mod can call Load.SetMapperMods() and erase what an earlier
// one added, while the JOIN and extra SELECT columns it introduced stay on
// the build copy — an end-of-chain snapshot would see zero and let the
// original P-06 failure ("the requested relation is silently missing")
// through (P-06 Codex review 3, Important).
//
// A parent that exposes neither Apply(...bob.Mod[*dialect.SelectQuery]) nor
// its inner *dialect.SelectQuery is NOT assumed safe: an outer wrapper that
// only embeds bob.Query still forwards WriteQuery to an inner query whose
// contextual mods do run, so skipping the check would be a silent bypass.
// Only bob.BaseQuery[expr.Clause] (the exact type psql.RawQuery returns,
// whose expression carries no ContextualModdable at all) is allowed
// through; everything else is rejected with *UncheckableParentQueryError.
// This reverses deviation 1 recorded in 12-プラン §2 (P-06 round 3).
func checkMapperModsAfterContextualMod(
	mod bob.ContextualMod[*dialect.SelectQuery],
	declared []declaredCountPreload,
) bob.ContextualMod[*dialect.SelectQuery] {
	return bob.ContextualModFunc[*dialect.SelectQuery](
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			nextCtx, err := mod.Apply(ctx, built)
			if err != nil {
				return nextCtx, err
			}

			n := len(built.GetMapperMods())
			// P-13: same 1:1 判定 as checkMapperMods above(防御強化案(a)、
			// Codexレビュー1回目、2026-08-20)。built.PreloadColumnsは
			// built(実際にビルドされた*dialect.SelectQuery)自身の
			// clause.SelectList埋め込みフィールドなので、ここではdeclared
			// を読み取った時点のスナップショットではなく、この時点での
			// 実際の総数を都度計算する。
			if n == 0 || (n == len(declared) && len(built.PreloadColumns) == len(declared)) {
				return nextCtx, nil
			}
			return nextCtx, &UnsupportedMapperModsError{Count: n}
		},
	)
}

func installBuildTimeMapperModCheck(
	parentQuery bob.Query,
	declared []declaredCountPreload,
) (func(), error) {
	moddable, ok := parentQuery.(psqlSelectModdable)
	if !ok {
		switch parentQuery.(type) {
		case bob.BaseQuery[expr.Clause], *bob.BaseQuery[expr.Clause]:
			return func() {}, nil
		default:
			return nil, &UncheckableParentQueryError{
				Type: fmt.Sprintf("%T", parentQuery),
			}
		}
	}

	var selectQuery *dialect.SelectQuery
	moddable.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	if selectQuery == nil {
		return nil, &UncheckableParentQueryError{
			Type: fmt.Sprintf("%T", parentQuery),
		}
	}

	originalMods := selectQuery.ContextualModdable.Mods
	checkedMods := make(
		[]bob.ContextualMod[*dialect.SelectQuery],
		0,
		len(originalMods),
	)
	for _, mod := range originalMods {
		checkedMods = append(
			checkedMods,
			checkMapperModsAfterContextualMod(mod, declared),
		)
	}
	selectQuery.ContextualModdable.Mods = checkedMods

	return func() {
		selectQuery.ContextualModdable.Mods = originalMods
	}, nil
}

// parentOrderExprRe matches a single ORDER BY expression that resolves to
// "<table>"."<column>", optionally followed by ASC/DESC and NULLS
// FIRST/LAST — the shape outerParentOrd requires every expression to have
// before it will move row_number() outside the parent subquery.
var parentOrderExprRe = regexp.MustCompile(
	`^"([^"]+)"\."([^"]+)"( ASC| DESC)?( NULLS FIRST| NULLS LAST)?$`,
)

// outerParentOrd decides whether composeIsland should number _sl_src rows
// with row_number() placed OUTSIDE the parent subquery (see this file's
// installParentOrdColumn doc for why the inside placement can defeat a
// LIMIT: PostgreSQL's Limit sits above WindowAgg in the inside plan, so the
// planner cannot push it below the window and must sort every parent row
// before truncating). Moving row_number() outside works only when it can
// ORDER BY the exact same expressions the parent already ORDERs BY —
// column references need no rewriting when they are the parent table's own
// columns, referenced positionally by the outer wrapper's alias — so this
// function accepts a parent query only when ALL of the following hold:
//
//   - the query is a bare table (parts.Schema == "", parts.Table != ""):
//     the outer wrapper's FROM-alias can equal the table name, matching
//     what the ORDER BY expressions already reference.
//   - it exposes *dialect.SelectQuery (psqlSelectModdable).
//   - it has at least one ORDER BY expression.
//   - it is not DISTINCT/DISTINCT ON and does not combine queries
//     (UNION/INTERSECT/EXCEPT) — same opt-out as installParentOrdColumn.
//   - it has a LIMIT (Limit.Count != nil): without one, the inside
//     placement already gets the same plan a LIMIT-less outer wrapper
//     would, so there is nothing to gain and installParentOrdColumn's
//     established path is kept.
//   - every ORDER BY expression, rendered alone, is a bare
//     "<table>"."<column>" reference (optionally ASC/DESC and NULLS
//     FIRST/LAST) to parts.Table itself, with no bind parameters — an
//     expression touching a joined table cannot be resolved against the
//     outer wrapper's single-table alias.
//
// Any other shape falls back to the caller using installParentOrdColumn
// instead; this function does not guess.
//
// gpt-review LOW (P-39): before assigning a hoisted key's name
// ("_sl_row_sort<i>", sortColPrefix), outerParentOrd checks it against
// cols (TableParts.Columns, the parent table's own column names). If the
// name is already taken by a real parent column, the outer wrapper's
// unqualified alias reference would either duplicate that column in the
// parent's SELECT list (making the outer window's reference to it
// ambiguous) or silently resolve to the wrong column, so this function
// declines the whole query (fail-closed) rather than pick a different
// name; the caller falls back to installParentOrdColumn's inside
// placement. This check only ever looks at TableParts.Columns — it does
// not inspect the parent SELECT's own expressions for an explicit alias
// that happens to collide, because fvuls's generated code never attaches
// any alias other than a parent column's own name to a parent query's
// SELECT list.
//
// P-39: an ORDER BY expression no longer has to be a bare reference to the
// parent table's own column for outerParentOrd to accept the query. Each
// element is now classified independently (bareParentColumnOrder): a bare
// "<table>"."<column>" reference is reused verbatim (byte-identical SQL,
// issue P-39 §5.3), while anything else (a joined table's column, an
// EXISTS subquery, a computed expression) is hoisted onto the parent's own
// SELECT list under an alias ("_sl_row_sort<i>", sortColPrefix) — see
// hoistParentSortKeys below — and the outer window's ORDER BY references
// that alias instead. hoisted carries, in order, the original Expression
// field of every clause.OrderDef this function decided to hoist; the
// caller is responsible for actually appending those hoisted columns via
// hoistParentSortKeys.
func outerParentOrd(
	ctx context.Context,
	parentQuery bob.Query,
	schema, table string,
	cols []Column,
) (orderExprs []bob.Expression, hoisted []any, alias string, ok bool) {
	if schema != "" || table == "" {
		return nil, nil, "", false
	}

	moddable, isModdable := parentQuery.(psqlSelectModdable)
	if !isModdable {
		return nil, nil, "", false
	}

	var selectQuery *dialect.SelectQuery
	moddable.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	if selectQuery == nil || len(selectQuery.OrderBy.Expressions) == 0 {
		return nil, nil, "", false
	}
	if selectQuery.Distinct.On != nil || len(selectQuery.Combines.Queries) > 0 {
		return nil, nil, "", false
	}
	if selectQuery.Limit.Count == nil {
		return nil, nil, "", false
	}

	// Snapshotted before any further inspection: a later contextual mod in
	// the chain (there is none yet at this call site, but the same
	// precaution installParentOrdColumn takes applies) could otherwise
	// consume/clear the parent's own OrderBy before it is read again.
	candidates := append([]bob.Expression(nil), selectQuery.OrderBy.Expressions...)

	colNames := make(map[string]struct{}, len(cols))
	for _, c := range cols {
		colNames[c.Name] = struct{}{}
	}

	outer := make([]bob.Expression, 0, len(candidates))
	for _, e := range candidates {
		// A bare reference to the parent table's own column: keep it
		// exactly as-is, so a parent whose ORDER BY is entirely made of
		// such references still produces byte-identical SQL to before
		// P-39 (issue P-39 §5.3).
		if bareParentColumnOrder(ctx, e, table, colNames) {
			outer = append(outer, e)
			continue
		}

		// Anything else must be hoisted. Direction/Nulls/Collation need to
		// be split off from Expression, which requires the element to be
		// a clause.OrderDef value (issue P-39 §2.6) — anything else (a
		// bare string is still possible via sm.OrderBy("...") without
		// .Desc()/.NullsLast(), but such a plain string would already have
		// matched bareParentColumnOrder or been unparseable above) is not
		// something this function knows how to split, so it declines the
		// whole query rather than guess at direction (issue P-39 §6.2).
		od, isOrderDef := e.(clause.OrderDef)
		if !isOrderDef {
			return nil, nil, "", false
		}
		name := sortColPrefix + strconv.Itoa(len(hoisted))
		if _, collides := colNames[name]; collides {
			// gpt-review LOW (P-39): the parent table already has a real
			// column named exactly what this hoisted key would be
			// called, so hoisting under this name is unsafe (see this
			// function's doc comment). Bail out of the whole outer
			// placement rather than guess at a different name.
			return nil, nil, "", false
		}
		hoisted = append(hoisted, od.Expression)
		outer = append(outer, clause.OrderDef{
			Expression: psql.Quote(table, name),
			Direction:  od.Direction,
			Nulls:      od.Nulls,
			Collation:  od.Collation,
		})
	}

	return outer, hoisted, table, true
}

// bareParentColumnOrder reports whether one ORDER BY element renders as a
// bare "<table>"."<column>" reference (optionally ASC/DESC and NULLS
// FIRST/LAST) to table itself, with no bind parameters — the shape the
// outer wrapper's single-table alias can resolve without any rewriting.
// This is outerParentOrd's pre-P-39 whole-query condition, now applied per
// element (issue P-39 §6.1).
func bareParentColumnOrder(ctx context.Context, e bob.Expression, table string, colNames map[string]struct{}) bool {
	var buf bytes.Buffer
	args, err := bob.Express(ctx, &buf, dialect.Dialect, 1, e)
	if err != nil || len(args) != 0 {
		return false
	}
	m := parentOrderExprRe.FindStringSubmatch(buf.String())
	if m == nil {
		return false
	}
	if m[1] != table {
		return false
	}
	_, known := colNames[m[2]]
	return known
}

// aliasedSortKey renders "<expr> AS \"<alias>\"" for one hoisted ORDER BY
// expression. clause.OrderDef.Expression is an `any` — it may be a
// bob.Expression (dbmodels.X.Columns.Y, psql.Exists(...)) or a bare string
// (sm.OrderBy("id")) — so this cannot go through
// dialect.NewExpression(...).As(...), which requires a bob.Expression.
// bob.Express handles both shapes (issue P-39 §7 Task 3).
type aliasedSortKey struct {
	expr  any
	alias string
}

func (a aliasedSortKey) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.Express(ctx, w, d, start, a.expr)
	if err != nil {
		return nil, err
	}
	if _, err := w.WriteString(" AS "); err != nil {
		return nil, err
	}
	d.WriteQuoted(w, a.alias)
	return args, nil
}

// hoistParentSortKeys appends, as the LAST contextual mod of the parent
// query, one aliased SELECT column per expression in exprs, named
// sortColPrefix+"0", sortColPrefix+"1", ... in the same order
// outerParentOrd assigned them. The outer _sl_src wrapper's row_number()
// ORDER BY references those aliases instead of the original expressions,
// so an ORDER BY touching a joined table or an EXISTS subquery can still
// use the outer placement (issue P-39 §6.1) — and the parent keeps its own
// ORDER BY and LIMIT untouched, so its top-N sort survives.
//
// The restore func trims the mod back off so a second All/Build on the
// same query does not add the columns twice, matching
// installParentOrdColumn's own contract below.
func hoistParentSortKeys(parentQuery bob.Query, exprs []any) func() {
	if len(exprs) == 0 {
		return func() {}
	}
	moddable, ok := parentQuery.(psqlSelectModdable)
	if !ok {
		return func() {}
	}

	var selectQuery *dialect.SelectQuery
	moddable.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	if selectQuery == nil {
		return func() {}
	}

	originalMods := selectQuery.ContextualModdable.Mods
	selectQuery.AppendContextualModFunc(func(ctx context.Context, built *dialect.SelectQuery) (context.Context, error) {
		for i, e := range exprs {
			built.SelectList.AppendSelect(aliasedSortKey{expr: e, alias: sortColPrefix + strconv.Itoa(i)})
		}
		return ctx, nil
	})

	return func() {
		selectQuery.ContextualModdable.Mods = originalMods
	}
}

// installParentOrdColumn appends, as the LAST contextual mod of the parent
// query, a row_number() window ordered by the parent's own ORDER BY
// expressions, aliased ordCol, so composeIsland can number _sl_src rows by
// it instead of relying on materialization order (issue: Copilot review on
// PR #1997, compose.go:704 — a top-level parentQuery's row_number() OVER ()
// has no SQL-level guarantee of following the parent's ORDER BY).
//
// Returns "" (and a no-op restore) when the parent has no ORDER BY, is
// DISTINCT/DISTINCT ON, or combines queries (UNION/INTERSECT/EXCEPT): none
// of those are seen in practice for a sectionload parent, and the window
// function's placement relative to DISTINCT/a set operation is not
// something this helper reasons about, so it opts out rather than guess.
// The restore func trims the mod back off so a second All/Build on the
// same query does not add the column twice.
func installParentOrdColumn(parentQuery bob.Query) (string, func()) {
	moddable, ok := parentQuery.(psqlSelectModdable)
	if !ok {
		return "", func() {}
	}

	var selectQuery *dialect.SelectQuery
	moddable.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	if selectQuery == nil || len(selectQuery.OrderBy.Expressions) == 0 {
		return "", func() {}
	}
	if selectQuery.Distinct.On != nil || len(selectQuery.Combines.Queries) > 0 {
		return "", func() {}
	}

	// Copied out of selectQuery now: the contextual mod below runs against
	// a build-time copy (built), not selectQuery itself, and by then the
	// parent's own OrderBy may already have been consumed/cleared by an
	// earlier mod in the chain.
	orderExprs := append([]bob.Expression(nil), selectQuery.OrderBy.Expressions...)

	originalMods := selectQuery.ContextualModdable.Mods
	selectQuery.AppendContextualModFunc(func(ctx context.Context, built *dialect.SelectQuery) (context.Context, error) {
		winMods := make([]bob.Mod[*clause.Window], 0, len(orderExprs))
		for _, e := range orderExprs {
			winMods = append(winMods, bob.ModFunc[*clause.Window](func(w *clause.Window) {
				w.AppendOrder(e)
			}))
		}
		built.SelectList.AppendSelect(psql.F("row_number")(fm.Over(winMods...)).As(ordCol))
		return ctx, nil
	})

	return ordCol, func() {
		selectQuery.ContextualModdable.Mods = originalMods
	}
}

// All executes a sectioned load: one UNION ALL query for this island,
// plus one query per Then(...) boundary. The result slice preserves the
// parent query's row order.
//
// 親クエリは、対象テーブルの全列を選択する必要がある。
// 親列サブセットは未対応で、非 nil エラーを返す。具体的なエラー型は
// 保証しない。結果列の検査より前に、PostgreSQL が合成SQLを拒否する場合がある。
// 列がずれた状態でモデルを走査することはない。
func All[T any](
	ctx context.Context,
	exec bob.Executor,
	parentQuery bob.Query,
	parts TableParts[T],
	nodes ...Node,
) ([]T, error) {
	// [0] Parent query hooks, in the same order bob.All would run them
	// (e.g. tenancy conditions land inside _sl_src). Must run before plan
	// validation below: bob's hook-first contract (P-06) requires these
	// hooks to always run, even when the declared tree itself turns out
	// to be invalid (P-03 round 2 Minor 2, Codex review 2026-08-09).
	if h, ok := parentQuery.(bob.HookableQuery); ok {
		var err error
		ctx, err = h.RunHooks(ctx, exec)
		if err != nil {
			return nil, err
		}
	}

	// [0.5] P-33: resolve each edge's caller-provided Only/Except column
	// selection into a concrete ColumnPlan, on a copy of that edge — see
	// columnselect.go's resolveColumnPlans doc. Must run after hooks
	// above (P-06's hook-first contract) and before validatePlan below
	// (which needs to see the resolved ColumnPlan to require the
	// matching *KeyFromBuf overrides).
	var err error
	nodes, err = resolveColumnPlans(nodes)
	if err != nil {
		return nil, err
	}

	// [0.6] P-13: parentQuery自身のPreloadColumnsからbob PreloadCount宣言を
	// 読み取る。All はlenient=falseで呼ぶ: 宣言を読み取れない親クエリ型は
	// 従来どおり *UncheckableParentQueryError で拒否する(この判定は[1.6]の
	// installBuildTimeMapperModCheckが既に同じ型に対して行っているものと
	// 同じであり、二重に見えても実害は無い)。
	declared, totalPreloadColumns, err := declaredCountPreloads(ctx, parentQuery, false)
	if err != nil {
		return nil, err
	}

	// [1] Static plan validation: after the parent query hooks above and
	// before sectionload builds or executes its own query (P-06方式と同型)。
	// Catches a missing *KeyFromBuf override or an incomplete nested
	// ColumnPlan before the sectioned query runs. A hook receives exec and
	// may itself run SQL, so this does not claim to precede arbitrary SQL.
	// It needs only the declared tree, so it does not depend on the hooks
	// above, but it must still come after them per the ordering rule
	// stated there. Must also come before [1.5]/[1.6] below (issue P-06's
	// documented P-02 integration ordering: hooks → validatePlan → the
	// MapperMod checks) — P-03's validatePlan-driven errors (e.g.
	// MissingKeyFromBufError) must win over P-06's MapperMod rejection
	// when both would fire for the same call.
	if err := validatePlan(nodes, declared); err != nil {
		return nil, err
	}

	// [1.5] Reject a parent query carrying mapper mods sectionload cannot
	// apply (issue P-06). Checked after hooks, not before: a
	// bob.SelectQueryHooks hook receives the same mutable *SelectQuery a
	// Preload mod would (bob's orm.ExecQuery.RunHooks passes it straight
	// through), so a hook *could* add a mapper mod via
	// Preloader.Apply/AppendMapperMod that wasn't there before hooks ran
	// (issue P-06 A-06-1) — no hook in this codebase does that today, but
	// checking after hooks costs nothing and covers the case if one ever
	// does. When [1.5] itself rejects, sectionload's composed query (below,
	// [2]) has not been built yet — but this says nothing about [0]'s
	// hooks, which already ran by this point and receive a real
	// bob.Executor (bob.Hook's signature), so a hook that itself calls
	// QueryContext/ExecContext could already have reached the database
	// before this check ever runs (P-06 Minor, Codex review 2026-08-09: an
	// earlier version of this comment claimed "no query reaches the
	// database" for a rejection here, which overstated what's actually
	// guaranteed — no hook in this codebase has such a side effect today,
	// but the runtime does not enforce that). Build (below) intentionally
	// does not call this: it exists to let a caller see the SQL
	// sectionload would run, which never applies mapper mods regardless
	// (issue P-06 A-06-3).
	if err := checkMapperMods(parentQuery, declared, totalPreloadColumns); err != nil {
		return nil, err
	}

	// [1.6] Guard the gap [1.5] cannot see: a contextual mod (e.g. a real
	// Preloader.Apply run through AppendContextualModFunc) can add a
	// MapperMod after [1.5]'s snapshot. Wrap every contextual mod so the
	// check runs right after each one fires, and reject a parent query
	// whose type makes the check impossible (P-06 round 3). Since P-17
	// removed the separate pre-Build that used to render the parent alone,
	// the wrapped mods now fire while composeIsland's composed query is
	// itself being built ([2]/execIsland's bob.Build) — a [1.6] rejection
	// therefore happens partway through that build, not before it starts
	// (P-06×P-17 interaction, Codex review 2026-08-13). Both [1.5] and
	// [1.6] still return before execIsland's QueryContext/ExecContext, so
	// neither ever reaches the database.
	restoreMapperModCheck, err := installBuildTimeMapperModCheck(parentQuery, declared)
	if err != nil {
		return nil, err
	}
	defer restoreMapperModCheck()

	// [1.7] Give _sl_src's row_number() something to ORDER BY when the
	// caller's parentQuery has its own ORDER BY, instead of leaving it to
	// rely on materialization order (Copilot review, PR #1997,
	// compose.go:704). outerParentOrd is tried first: when the parent has
	// a LIMIT and ORDER BY expressions naming only its own columns, moving
	// row_number() to the outside of the parent subquery lets the LIMIT
	// run first (see outerParentOrd's doc comment). Anything else falls
	// back to installParentOrdColumn's inside placement, unchanged.
	outerOrd, hoistedSortKeys, outerAlias, useOuter := outerParentOrd(ctx, parentQuery, parts.Schema, parts.Table, parts.Columns)
	parentOrdCol := ""
	restoreOrd := func() {}
	if useOuter {
		restoreOrd = hoistParentSortKeys(parentQuery, hoistedSortKeys)
	} else {
		parentOrdCol, restoreOrd = installParentOrdColumn(parentQuery)
	}
	defer restoreOrd()

	// [2] Plan, compose, execute, dispatch. The reserved-name collision
	// check now happens inside composeIsland's parentSource wrapper
	// (parentexpr.go), at the same moment the parent query is written
	// into _sl_src — not as a separate pre-Build here, which used to
	// build (and re-run the parent's ContextualMods) a second time.
	plan, err := planIsland(nodes)
	if err != nil {
		return nil, err
	}
	plan.declaredCounts = declared
	plan.parentOuterOrd, plan.parentOuterAlias = outerOrd, outerAlias
	root := partsAdapter[T]{parts}
	q, err := composeIsland(parentQuery, root, plan, DefaultMaxColumns, parentOrdCol)
	if err != nil {
		return nil, err
	}
	res, err := execIsland(ctx, exec, q, root, plan, nil)
	if err != nil {
		return nil, err
	}

	// [3] Stitch bottom-up.
	if err := stitchIsland(plan, res); err != nil {
		return nil, err
	}

	// [4] Restore the parent query's row order via the ordinality column.
	parentsAny := res.items[0]
	idx := make([]int, len(parentsAny))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return res.ords[idx[a]] < res.ords[idx[b]] })

	orderedParents := make([]any, len(parentsAny))
	for i, j := range idx {
		orderedParents[i] = parentsAny[j]
	}
	res.items[0] = orderedParents

	// P-28 §6/§7.1 sync invariant: items[0][i] ↔ thenKeys[tp.id][i].
	// Reordering the parents by _sl_ord therefore must reorder every
	// root-level Then's key column with the same index permutation
	// (P-03 round 2 Critical 2) — runThen consumes parentKeyVals[i]
	// positionally against parents[i] (hooks.go).
	for _, tp := range plan.thens {
		if tp.parentSet != 0 {
			continue
		}
		keys := res.thenKeys[tp.id]
		if len(keys) != len(idx) {
			return nil, fmt.Errorf(
				"sectionload: %s: %d then parent keys for %d parents",
				tp.edge.relName(), len(keys), len(idx),
			)
		}
		orderedKeys := make([]any, len(keys))
		for i, j := range idx {
			orderedKeys[i] = keys[j]
		}
		res.thenKeys[tp.id] = orderedKeys
	}

	out := make([]T, len(orderedParents))
	for i, p := range orderedParents {
		out[i] = p.(T)
	}

	// [5] Hoist loaders attached to the parent query (e.g. SelectThenLoad
	// mods). Embedding does not propagate them to the outer query, so run
	// them explicitly.
	//
	// bob.Allx runs the loaders first and the after-query hook last
	// (bob@v0.49.0 exec.go:178-190). Keep this block before [6] and keep the
	// order in sync with bob.Allx.
	if l, ok := parentQuery.(bob.Loadable); ok {
		var retrieved any = out
		if parts.AsSlice != nil {
			retrieved = parts.AsSlice(out)
		}
		for _, ld := range l.GetLoaders() {
			if err := ld.Load(ctx, exec, retrieved); err != nil {
				return nil, err
			}
		}
	}

	// [6] Then boundaries: one recursive island each.
	//
	// Then() is documented as equivalent to ThenLoad, and bob completes
	// ThenLoad's loaders before AfterQueryHook fires (bob@v0.49.0
	// exec.go:178-190). This block must therefore run before the parent's
	// after-select hook, so a hook inspecting R fields loaded via Then(...)
	// sees a complete graph (issue P-24).
	for _, tp := range plan.thens {
		if err := runThen(ctx, exec, tp, res.items[tp.parentSet], res.thenKeys[tp.id]); err != nil {
			return nil, err
		}
	}

	// [7] Parent after-select hooks run for the parent results only.
	// Runs last, after [5]'s loaders and [6]'s Then boundaries, mirroring
	// bob.Allx's AfterQueryHook call at exec.go:186-190.
	if parts.AsSlice != nil {
		if h, ok := parts.AsSlice(out).(bob.HookableType); ok {
			if err := h.AfterQueryHook(ctx, exec, bob.QueryTypeSelect); err != nil {
				return nil, err
			}
		}
	}

	return out, nil
}

// BuiltQuery is one island's SQL and arguments, for inspection/debugging.
type BuiltQuery struct {
	SQL  string
	Args []any
}

// Build renders the FIRST island's query without executing it (the
// bob.Debug-style observability hook). Then(...) islands depend on key
// values collected at runtime and are therefore not included.
//
// Unlike All, Build does not reject a parentQuery carrying mapper mods
// (issue P-06 A-06-3): it exists to let a caller inspect the SQL
// sectionload would actually run, and that SQL never applies mapper mods
// regardless of whether this check runs — parentQuery is embedded
// unmodified inside _sl_src either way. Calling All with the same
// parentQuery still rejects it.
//
// P-13: for the same reason, Build reads declared count preloads
// leniently (declaredCountPreloads(..., lenient: true)) — a parentQuery
// type it cannot introspect is treated as zero declarations rather than
// rejected, so this function's A-06-3 contract (never reject a parent
// query type on Build's account) is not narrowed by P-13's addition.
func Build[T any](
	ctx context.Context,
	parentQuery bob.Query,
	parts TableParts[T],
	nodes ...Node,
) (BuiltQuery, error) {
	// [P-33] see All()'s [0.5] comment. Build has no hooks step to order
	// against, so this simply runs first.
	nodes, err := resolveColumnPlans(nodes)
	if err != nil {
		return BuiltQuery{}, err
	}

	// P-13: Build は MapperMod 検査をしない(A-06-3、変更なし)が、
	// composeIsland に渡す declaredCounts の threading は All と揃える
	// 必要がある — さもないと Build が返す SQL と All が実行する SQL が
	// 同じ parentQuery に対して食い違う。
	declared, _, err := declaredCountPreloads(ctx, parentQuery, true)
	if err != nil {
		return BuiltQuery{}, err
	}

	// Same as All's [1.7]: give _sl_src's row_number() the parent's own
	// ORDER BY to follow, when it has one, preferring outerParentOrd's
	// outside placement over installParentOrdColumn's inside one under the
	// same conditions. Build has no MapperMod check to order against, so
	// restoreOrd is the only cleanup this function needs before returning.
	outerOrd, hoistedSortKeys, outerAlias, useOuter := outerParentOrd(ctx, parentQuery, parts.Schema, parts.Table, parts.Columns)
	parentOrdCol := ""
	restoreOrd := func() {}
	if useOuter {
		restoreOrd = hoistParentSortKeys(parentQuery, hoistedSortKeys)
	} else {
		parentOrdCol, restoreOrd = installParentOrdColumn(parentQuery)
	}
	defer restoreOrd()

	plan, err := planIsland(nodes)
	if err != nil {
		return BuiltQuery{}, err
	}
	plan.declaredCounts = declared
	plan.parentOuterOrd, plan.parentOuterAlias = outerOrd, outerAlias
	root := partsAdapter[T]{parts}
	q, err := composeIsland(parentQuery, root, plan, DefaultMaxColumns, parentOrdCol)
	if err != nil {
		return BuiltQuery{}, err
	}
	sqlStr, args, err := bob.Build(ctx, q)
	if err != nil {
		return BuiltQuery{}, err
	}
	return BuiltQuery{SQL: sqlStr, Args: args}, nil
}
