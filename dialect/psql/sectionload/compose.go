package sectionload

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/fm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
)

// Reserved identifiers. The parent query must not use these names —
// parentSource (parentexpr.go) checks for them while composeIsland's
// _sl_src wrapper is itself being built, not before composing starts;
// violation is an explicit error.
const (
	colSet = "_sl_set"
	colOrd = "_sl_ord"
	cteSrc = "_sl_src"
	// srcAlias is the alias of each section's row source inside its own
	// branch. Branches are independent SELECTs, so one constant is fine.
	srcAlias = "_sl_t"
	// ordDBType is colOrd's own SQL type wherever a branch NULL-pads it
	// instead of reading a real ordinal: row_number() OVER () (what
	// ordered branches actually read back) returns bigint in PostgreSQL.
	// A count branch's NULL padding happened to reuse countDBType (also
	// "bigint") instead of this — harmless today, but the two types have
	// no reason to stay equal, so colOrd gets its own constant (Copilot
	// review, 2026-08-14).
	ordDBType = "bigint"
)

// countPreloadDBType is __count_<name> columns' own SQL type wherever a
// branch NULL-pads them: bob's countPreloader always wraps count(*) in a
// correlated subquery (bob_counts.bob.go.tpl), which PostgreSQL returns
// as bigint — the same value countDBType (node.go, P-04's own count
// sections) happens to hold today, but the two have no reason to stay
// equal, so this gets its own constant (same reasoning as ordDBType,
// compose.go:29-36).
const countPreloadDBType = "bigint"

// DefaultMaxColumns is the column-count guardrail. PostgreSQL
// caps a result row at 1664 columns; exceeding the threshold is an
// explicit error, fixable by splitting the tree with Then(...).
const DefaultMaxColumns = 1500

func cteName(set int) string {
	if set == 0 {
		return cteSrc
	}
	return fmt.Sprintf("_sl_n%d", set)
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// tableExpr renders a table name as a bob expression, reproducing bob's own
// NameExpr priority (dialect/psql/view.go) exactly (P-12): a non-empty
// generated schema is always fixed and never overridden by a runtime
// psql.UseSchema(ctx, ...) context value, matching how bob's own generated
// views behave. A generated schema of "" (fvuls: 129/129 tables today, per
// P-12's fact table) delegates to orm.SchemaTable, which reads
// orm.CtxUseSchema from the query-build context at WriteSQL time — the same
// mechanism the parent query already uses, so the parent and every relation
// sectionload composes agree on which schema they read from.
func tableExpr(schema, table string) bob.Expression {
	if schema == "" {
		return orm.SchemaTable(table)
	}
	return psql.Quote(schema, table)
}

// hopAliasPrefix/hopAlias name the intermediate table(s) a multi-hop edge
// (P-05) JOINs through. Scoped to one edge's own subquery/CTE (never the
// outer island layout), so — unlike _sl_set/_sl_ord/_sl_src/_sl_n<k>/
// _sl_cnt<k> — these never need reservedNameRe protection (P-05 §13.6(3)).
const hopAliasPrefix = "_sl_h"

func hopAlias(i int) string { return fmt.Sprintf("%s%d", hopAliasPrefix, i) }

// sideToAlias returns the alias Sides[i].To is read under: the final
// table's own alias (srcAlias) when i is the last side, otherwise the
// NEXT hop's alias (that hop's From IS this side's To — bob's own
// Relationship.Validate() guarantees Sides[i].To == Sides[i+1].From, see
// parts.go's Side doc). For a single-hop edge (n==1), sideToAlias(0, 1)
// is srcAlias — the pre-P-05 alias every existing call site already uses,
// which is exactly why matchAlias needs no separate single-hop branch
// (P-05 §13.6(3)).
func sideToAlias(i, n int) string {
	if i == n-1 {
		return srcAlias
	}
	return hopAlias(i + 1)
}

// sectionAliasMap (P-37) maps each table name this section's own SELECT
// reads to the alias it is read under inside that SELECT. A caller mod
// that qualifies a column with the table's own name (which every
// dbmodels.<Table>.Columns.<Col> expression does) can then be rewritten
// to the alias the FROM clause actually declares.
//
// Single-hop: e.child()'s own table -> srcAlias.
// Multi-hop: e.sides()[i].ToTable -> sideToAlias(i, len(sides)) for every
// i, which is srcAlias for the last side and hopAlias(i+1) otherwise.
// Route 3 (the section re-reads its own CTE) gets the same child-table
// entry: the CTE exposes the child's columns under their own names, so
// "<child>"."<col>" -> "_sl_t"."<col>" is still the right rewrite.
//
// e.sides()[i].ToTable is "" for a pre-P-05 single-hop Edge/CountEdge that
// never set Sides (sides() synthesizes a length-1 Side from the flat
// FromCols/ToCols fields, which carry no table names) — that entry is
// skipped, leaving only the child-table entry, which is exactly what a
// single-hop section needs.
//
// A table name recorded at two DIFFERENT aliases (a self-referencing
// multi-hop edge that reads the same table under two different hop
// aliases) is ambiguous — a bare table-name qualifier cannot say which
// occurrence it means — so that name is removed from the map entirely
// (P-37 Important-1, gpt-review): a caller mod that qualifies with it is
// left unresolved and still fails loudly, the same way it did before this
// change existed, rather than silently resolving to whichever occurrence
// happened to be recorded last. The same name recorded twice at the SAME
// alias (e.g. the child table via both the explicit entry above and the
// last side's ToTable) is not ambiguous and is kept.
func sectionAliasMap(e edgeCore) map[string]string {
	sides := e.sides()
	m := make(map[string]string, len(sides)+1)
	ambiguous := make(map[string]bool, len(sides)+1)
	set := func(table, alias string) {
		if table == "" || ambiguous[table] {
			return
		}
		if existing, ok := m[table]; ok {
			if existing == alias {
				return
			}
			delete(m, table)
			ambiguous[table] = true
			return
		}
		m[table] = alias
	}

	set(e.child().tableName(), srcAlias)
	n := len(sides)
	for i := 0; i < n; i++ {
		set(sides[i].ToTable, sideToAlias(i, n))
	}
	return m
}

// hopJoin is one INNER JOIN of a multi-hop edge's chain: alias for the
// intermediate table (from), and its ON condition(s) against the
// already-aliased table it links to.
type hopJoin struct {
	alias string
	from  bob.Expression
	on    []bob.Expression
}

// carriedCol is one entry of a multi-hop edge's carried-key SELECT list:
// "<alias>"."<column>" AS "<as>" (P-05 §13.6(2)/(3)).
type carriedCol struct {
	alias  string
	column string
	as     string
}

// edgeJoinChain builds the JOIN chain, carried-key SELECT list, and match
// alias for one edge's PHYSICAL SELECT — routes 1, 2, and 4 of
// buildEdgeSelect (route 3 re-reads an already-joined CTE by name and
// needs none of this; see unionBranch). For a single-hop edge
// (len(sides())<=1) this returns (nil, nil, srcAlias): no JOINs, no
// carried columns, and the same match alias every pre-P-05 route already
// used — so buildEdgeSelect renders byte-identical SQL to before this
// function existed (P-05 §13.5-1, pinned by TestP05Baseline).
//
// Joins are walked from the LAST side backward to side 1 (mirroring bob's
// own 009_rel_query.go.tpl, which does the same "until len | reverse"):
// side i's ON condition references sideToAlias(i, n), which for i < n-1
// is a LATER side's own join alias (hopAlias(i+1)) — that alias must
// already exist in the FROM/JOIN chain by the time side i's JOIN is
// emitted, which is only guaranteed by emitting joins in descending i
// order.
//
// Side 0 is never turned into a JOIN: it is the semi-join/keys match
// against the ISLAND's parent (matchWhere/parentKeySubquery), not a table
// this edge's own SELECT reads directly. Its own fixed conditions
// (Sides[0].Where) are therefore handled entirely by
// matchWhere/parentKeySubquery via edgeCore.edgeWhere()/fromCols()/
// toCols() (which already resolve to sides()[0] — see node.go), not here.
// hopChainSpec (R-24) is the minimal interface edgeJoinChain actually
// needs: sides() and carriedCols(). edgeSpec (row edges) and countEdgeSpec
// (count edges, R-24) both satisfy it structurally, so every existing call
// site (unionBranch, composeIsland) that already passes an edgeSpec value
// keeps compiling unmodified, and countBranch (below) can now pass a
// countEdgeSpec value too — the same JOIN-chain builder reused verbatim
// for a multi-hop count (§6.3 of P-34-multihop-count.md: SQL INNER JOIN is
// commutative, so FROM-last-table-backward-JOIN produces the same result
// set as bob's own FROM-first-table-forward-JOIN LoadCount<Rel> shape).
type hopChainSpec interface {
	sides() []Side
	carriedCols() []Column
}

func edgeJoinChain(e hopChainSpec) (joins []hopJoin, carried []carriedCol, matchAlias string) {
	sides := e.sides()
	n := len(sides)
	matchAlias = sideToAlias(0, n)
	if n <= 1 {
		return nil, nil, matchAlias
	}

	joins = make([]hopJoin, 0, n-1)
	for i := n - 1; i >= 1; i-- {
		s := sides[i]
		fromAlias := hopAlias(i)
		toAlias := sideToAlias(i, n)
		on := make([]bob.Expression, 0, len(s.ToCols)+len(s.Where))
		for j, toCol := range s.ToCols {
			on = append(on, psql.Quote(toAlias, toCol).EQ(psql.Quote(fromAlias, s.FromCols[j])))
		}
		// This hop's OWN fixed condition(s) (Sides[i].Where, i>=1): unlike
		// side 0's, these never reach matchWhere/parentKeySubquery (those
		// only see edgeWhere()==sides()[0].Where) — they belong in this
		// JOIN's own ON clause, on whichever side of the hop they name
		// (P-05 §13.6(4)).
		for _, w := range s.Where {
			switch w.On {
			case "from":
				on = append(on, psql.Quote(fromAlias, w.Column).EQ(expr.Raw(w.SQLValue)))
			case "to":
				on = append(on, psql.Quote(toAlias, w.Column).EQ(expr.Raw(w.SQLValue)))
			}
		}
		joins = append(joins, hopJoin{
			alias: fromAlias,
			from:  tableExpr(s.FromSchema, s.FromTable),
			on:    on,
		})
	}

	// The carried key(s): Sides[0].ToCols, read from matchAlias (==
	// Sides[0].To's own alias, i.e. hopAlias(1) — the alias the i==1 join
	// above just assigned to that same table), aliased to their
	// SQL-layout names (Edge.Carried, in the same position — see
	// carriedCols()'s doc for why the names differ).
	side0 := sides[0]
	names := e.carriedCols()
	carried = make([]carriedCol, 0, len(side0.ToCols))
	for i, toCol := range side0.ToCols {
		as := toCol
		if i < len(names) {
			as = names[i].Name
		}
		carried = append(carried, carriedCol{alias: matchAlias, column: toCol, as: as})
	}
	return joins, carried, matchAlias
}

// edgeSelectSpec describes one relationship edge's row source for
// buildEdgeSelect: which columns to read, where to read them from, and how
// those rows are matched against the parent. All four relation-SELECT
// routes in composeIsland/runThen build one of these and delegate to the
// same builder (issue P-11 §11.3):
//
//  1. a same-island node's CTE body (its full row set, semi-joined against
//     its parent CTE) — semiFromCTE set;
//  2. a same-island leaf node's UNION ALL branch (table + semi-join) —
//     semiFromCTE set;
//  3. a same-island node-with-children's UNION ALL branch, read back from
//     its own CTE — neither semiFromCTE nor keys set, because the CTE body
//     (route 1) already applied the matching;
//  4. a Then(...) boundary's parent query, matched against an explicit,
//     already-collected list of parent key tuples — keys set.
type edgeSelectSpec struct {
	edge  edgeCore
	cols  []Column       // columns to SELECT, in Columns order (P-03 replaces this with a column plan)
	from  bob.Expression // table or CTE name expression to read from
	alias string         // alias assigned to `from` inside this SELECT

	// joins/carried/matchAlias (P-05): a multi-hop edge's JOIN chain
	// through its intermediate table(s), the carried-key SELECT list, and
	// the alias the semi-join/keys match and side-0 "to" fixed condition
	// are read against. All three come from edgeJoinChain, built by the
	// caller BEFORE this spec (edgeJoinChain needs edgeSpec, a strictly
	// larger interface than the edgeCore this struct embeds). Set at
	// routes 1, 2, and 4 (buildEdgeSelect's three PHYSICAL routes); left
	// empty/"" at route 3 (a with-children node's own CTE read-back),
	// which reads an already-joined row back by name and needs no JOIN of
	// its own — see unionBranch. joins/carried are nil and matchAlias ==
	// srcAlias for every single-hop edge, at every route, which is why
	// this addition renders byte-identical SQL to pre-P-05 (pinned by
	// TestP05Baseline).
	joins      []hopJoin
	carried    []carriedCol
	matchAlias string

	// Matching against the parent. At most one is set; neither set means
	// `from` needs no matching of its own (route 3 above).
	semiFromCTE string  // same-island: name of the parent CTE to semi-join against
	keys        [][]any // Then boundary: parent key tuples, compared by row-value IN

	// opts (issue P-02) carries the edge's own caller-provided SELECT mods.
	// Left zero at route 3 (a with-children node's UNION ALL branch, read
	// back from its own CTE): that CTE's body (route 1) already applied
	// opts once; applying again here would run WHERE/LIMIT/OFFSET twice.
	opts SelectOptions
	// collapseOrderBy is set at routes 1, 2, and 4 (this SELECT becomes one
	// arm of a UNION ALL, the body of a CTE read back by one, or a
	// Then(...) boundary's own parent query): a literal ORDER BY inside a
	// UNION ALL arm does not survive the union (issue P-02 §10.2), and a
	// route 4 query becomes the new island's own root, which composeIsland
	// wraps in its own order-restoring row_number() OVER () (_sl_src) the
	// same way every plain root query is — so both need the same rewrite
	// buildEdgeSelect applies here: any ORDER BY opts add is rewritten into
	// a row_number() OVER (ORDER BY ...) ordinal column instead, and the
	// literal ORDER BY is cleared. Left false only at route 3 (a
	// with-children node's UNION ALL branch, read back from its own CTE):
	// that CTE's body (route 1) already applied opts once, including this
	// same collapse, so route 3 must not repeat it.
	collapseOrderBy bool

	// countAs, when non-empty, turns this SELECT into a count section's
	// aggregate: count(*) is appended to the select list under that alias
	// (the layout's _sl_cnt<set> column, A-04-2), and groupBy names the
	// columns to GROUP BY — always the edge's own child-side join key, so
	// the result is exactly one row per parent key (issue P-04 §10.3
	// item 3). Empty at every row-section route.
	countAs string
	groupBy []Column

	// projection (P-36): when non-empty, this SELECT's column list
	// verbatim — the caller (unionBranch) has already laid out the whole
	// island's column order, including the NULL padding for every column
	// this branch does not own. cols/carried/countAs are then ignored.
	// Empty at routes 1 and 4 (a CTE body and a Then boundary's own parent
	// query are not UNION ALL arms, so they have no island layout to
	// follow), which keeps their SQL byte-identical.
	projection []any

	// fragment (P-36 gpt-review MEDIUM), when non-nil, is the
	// permittedFragment unionBranch already extracted from s.opts via
	// extractOptionsFragment — the SAME extraction unionBranchOrdColumn
	// used to compute the branch's _sl_ord row_number() ordinal. When set,
	// buildEdgeSelect reuses it instead of extracting s.opts a second
	// time, so the WHERE/ORDER BY actually applied to this SELECT and the
	// ordinal column unionBranchOrdColumn already emitted are guaranteed
	// to come from one and the same SelectOptions evaluation, even when
	// opts.mods carries a stateful bob.ModFunc (e.g. a counter) that
	// would otherwise render a different fragment on a second call. Left
	// nil at routes 1, 3, and 4 (route 3 passes no opts at all; routes 1
	// and 4 have no projection/ordinal of their own to keep in sync
	// with), where buildEdgeSelect falls back to extracting s.opts
	// itself, unchanged from before this field existed.
	fragment *permittedFragment
}

// ordCol is the name of the per-section row-ordinal column buildEdgeSelect
// adds to its own SELECT list when collapsing an ORDER BY opt (issue P-02
// §10.2). It is scoped to one edge's own subquery/CTE and never appears in
// the generated column contract, so it does not collide with _sl_ord (the
// root-restoring ordinal composeIsland already owns) or with any table
// column name.
const ordCol = "_sl_row_ord"

// sortColPrefix (P-39) names the columns outerParentOrd hoists onto the
// parent query's SELECT list when an ORDER BY expression cannot be
// referenced from the outer wrapper's single-table alias (a joined
// table's column, an EXISTS subquery, any computed expression). Like
// ordCol above, this name deliberately sits OUTSIDE reservedNameRe's
// namespace (parentexpr.go): the check exists to catch a CALLER's column
// colliding with sectionload's internal names, and these columns are
// sectionload's own additions to the parent's SELECT list, not the
// caller's. The i-th hoisted key is sortColPrefix + strconv.Itoa(i).
const sortColPrefix = "_sl_row_sort"

// buildEdgeSelect builds "SELECT <cols> FROM <from> AS <alias> [WHERE
// <match>]" for one edge's own rows, then merges in s.opts (issue P-02) on
// top. It never returns a partial result: the WHERE/JOIN structure lives
// entirely inside this function and matchWhere, so no caller needs to
// concatenate SQL text (issue P-11 §10 step 4).
//
// s.opts never touches this query directly — extractOptionsFragment (see
// options.go) applies it to an isolated, empty query instead, and only the
// permitted fragment (WHERE/ORDER BY/LIMIT/OFFSET/GROUP BY/HAVING) is
// merged in via permittedFragment.applyTo, after the join condition above.
// A caller mod therefore has no way to see, read, or delete the match
// condition sectionload itself depends on — not merely "applied, then
// checked for a diff" (issue P-02 Critical, Codex review 2026-08-09: the
// old apply-then-diff design never compared Where, so a mod that cleared
// it passed unnoticed).
func buildEdgeSelect(s edgeSelectSpec) (bob.BaseQuery[*pdialect.SelectQuery], error) {
	// R-24: a count section's own s.cols/s.groupBy (c.keyCols) name a
	// column of s.alias (the FINAL child table) only for a single-hop
	// count. For a multi-hop count, they name a column of the FIRST
	// INTERMEDIATE table instead — the same table edgeJoinChain's
	// matchAlias already points at — so SELECT/GROUP BY must be quoted
	// against matchAlias there, not s.alias. A row section never sets
	// countAs, so colAlias stays s.alias (unchanged) for every existing
	// call site (§6.4 of P-34-multihop-count.md). A single-hop count's own
	// matchAlias equals srcAlias (==s.alias here), so this branch renders
	// byte-identical SQL for every pre-R-24 count too.
	colAlias := s.alias
	if s.countAs != "" && s.matchAlias != "" {
		colAlias = s.matchAlias
	}

	var cols []any
	if len(s.projection) > 0 {
		// P-36: the caller (unionBranch) has already laid out the whole
		// island's column order, NULL padding included — use it verbatim
		// instead of building the column list from s.cols/s.carried/
		// s.countAs.
		cols = s.projection
	} else {
		cols = make([]any, 0, len(s.cols)+len(s.carried)+1)
		for _, c := range s.cols {
			cols = append(cols, psql.Quote(colAlias, c.Name))
		}
		// s.carried (P-05): a multi-hop edge's carried join-key column(s),
		// read from the intermediate table edgeJoinChain aliased and appended
		// AFTER s.cols — the same order childCols() concatenates them in
		// (childScanCols() then carriedCols()), which is what lets every
		// existing "read back column N of childCols()" call site (unionBranch,
		// section0List, expectedColumns, ...) keep working unmodified: it
		// never needs to know these came from a JOIN instead of the table
		// itself. Empty for a single-hop edge.
		for _, c := range s.carried {
			cols = append(cols, psql.Quote(c.alias, c.column).As(c.as))
		}
		if s.countAs != "" {
			cols = append(cols, psql.Raw("count(*)").As(s.countAs))
		}
	}

	mods := []bob.Mod[*pdialect.SelectQuery]{
		sm.Columns(cols...),
		sm.From(s.from).As(s.alias),
	}
	// s.joins (P-05): a multi-hop edge's intermediate-table JOIN chain,
	// in the order edgeJoinChain built it (last side first). Empty for a
	// single-hop edge, so this loop is a no-op and the FROM clause is
	// exactly what it was before P-05.
	for _, j := range s.joins {
		mods = append(mods, sm.InnerJoin(j.from).As(j.alias).On(j.on...))
	}
	if where := matchWhere(s); where != nil {
		mods = append(mods, sm.Where(where))
	}
	for _, c := range s.groupBy {
		mods = append(mods, sm.GroupBy(psql.Quote(colAlias, c.Name)))
	}

	q := psql.Select(mods...)
	if len(s.opts.mods) == 0 {
		return q, nil
	}

	var frag permittedFragment
	if s.fragment != nil {
		// P-36 gpt-review MEDIUM: unionBranch already extracted this
		// fragment once (to compute the branch's _sl_ord ordinal via
		// unionBranchOrdColumn) — reuse it instead of extracting s.opts a
		// second time, so both read the same WHERE/ORDER BY even when
		// opts.mods carries a stateful bob.ModFunc.
		frag = *s.fragment
	} else {
		var err error
		frag, err = extractOptionsFragment(s.opts, s.edge.relName())
		if err != nil {
			return bob.BaseQuery[*pdialect.SelectQuery]{}, err
		}
	}
	// P-37: resolve a caller mod's qualified column references (e.g.
	// dbmodels.<Table>.Columns.<Col>, which always renders
	// "<table>"."<col>") against the alias this section's own FROM
	// clause actually declares, before merging the fragment in.
	frag = rewriteQualifiedRefs(frag, sectionAliasMap(s.edge))
	frag.applyTo(q.Expression)

	if s.collapseOrderBy && len(q.Expression.OrderBy.Expressions) > 0 {
		if len(s.projection) == 0 {
			orderExprs := q.Expression.OrderBy.Expressions
			winMods := make([]bob.Mod[*clause.Window], 0, len(orderExprs))
			for _, e := range orderExprs {
				e := e
				winMods = append(winMods, bob.ModFunc[*clause.Window](func(w *clause.Window) {
					w.AppendOrder(e)
				}))
			}
			q.Expression.SelectList.AppendSelect(psql.F("row_number")(fm.Over(winMods...)).As(ordCol))
		}
		// projection 経路では、同じ row_number() 式を unionBranchOrdColumn が
		// _sl_ord の位置に既に埋めてある。ここでは ORDER BY を消すだけでよい。
		q.Expression.OrderBy.ClearOrderBy()
	}

	return q, nil
}

// matchWhere renders the match condition of an edgeSelectSpec: a row-value
// semi-join against a same-island parent CTE, a row-value IN over literal
// Then-boundary key tuples, or nil when the source needs no matching of its
// own (a with-children node re-reading its own already-filtered CTE).
// edgeWhere "to" conditions (fixed conditions on the child side) are ANDed
// in whenever a match condition is emitted at all; "from" conditions (on
// the parent side) apply only inside the row-value-IN subquery, via
// parentKeySubquery, matching the pre-P-11 row-value-IN condition exactly.
//
// The match columns (toCols()==sides()[0].ToCols) and side-0's own "to"
// fixed condition are read from s.matchAlias, not s.alias (P-05): for a
// multi-hop edge, sides()[0].To is the FIRST intermediate table, which
// edgeJoinChain aliases as hopAlias(1) — never s.alias (srcAlias, the
// FINAL table). s.matchAlias falls back to s.alias when empty. Every
// row-section route (1/2/3/4) sets matchAlias explicitly, and — since
// R-24 — countBranch's call also sets matchAlias (via edgeJoinChain, same
// as a row section), so this fallback is now a defensive default only,
// not a load-bearing path for any current caller.
func matchWhere(s edgeSelectSpec) bob.Expression {
	if s.semiFromCTE == "" && s.keys == nil {
		return nil
	}

	alias := s.matchAlias
	if alias == "" {
		alias = s.alias
	}

	to := make([]bob.Expression, 0, len(s.edge.toCols()))
	for _, c := range s.edge.toCols() {
		to = append(to, psql.Quote(alias, c))
	}

	var cond psql.Expression
	if s.semiFromCTE != "" {
		cond = psql.Group(to...).In(parentKeySubquery(s))
	} else {
		vals := make([]bob.Expression, 0, len(s.keys))
		for _, k := range s.keys {
			vals = append(vals, psql.ArgGroup(k...))
		}
		cond = psql.Group(to...).In(vals...)
	}

	var toWhere []bob.Expression
	for _, w := range s.edge.edgeWhere() {
		if w.On == "to" {
			toWhere = append(toWhere, psql.Quote(alias, w.Column).EQ(expr.Raw(w.SQLValue)))
		}
	}
	if len(toWhere) == 0 {
		return cond
	}
	return cond.And(toWhere...)
}

// parentKeySubquery builds the subquery used by matchWhere for a
// same-island semi-join: SELECT <from cols> FROM <parent CTE> [WHERE
// <from-where>]. edgeWhere "SQLValue" text is embedded via expr.Raw, which
// writes it verbatim without scanning it for "?" placeholders (P-09): a
// relationship condition like `data ? 'active'` (a jsonb operator) must not
// be miscounted as a bob argument placeholder the way psql.Raw/expr.Clause's
// textual "?" scanning would. Before this fix the miscount failed loudly at
// bob.Build time (an args/placeholder-count mismatch, expr/raw.go's
// rawError) rather than silently producing a wrong query — still worth
// fixing since a legal relationship configuration should not build-fail at
// all, but there is no silent-corruption risk to worry about here.
func parentKeySubquery(s edgeSelectSpec) bob.Expression {
	cols := make([]any, 0, len(s.edge.fromCols()))
	for _, c := range s.edge.fromCols() {
		cols = append(cols, psql.Quote(c))
	}

	mods := []bob.Mod[*pdialect.SelectQuery]{
		sm.Columns(cols...),
		sm.From(psql.Quote(s.semiFromCTE)),
	}

	var fromWhere []bob.Expression
	for _, w := range s.edge.edgeWhere() {
		if w.On == "from" {
			fromWhere = append(fromWhere, psql.Quote(w.Column).EQ(expr.Raw(w.SQLValue)))
		}
	}
	if len(fromWhere) > 0 {
		mods = append(mods, sm.Where(psql.And(fromWhere...)))
	}

	return psql.Select(mods...)
}

// ownSectionColumns (P-36) renders the columns this branch actually owns,
// in childCols() order: the child table's own columns read from srcAlias,
// then a multi-hop edge's carried key column(s) read from the intermediate
// table's own alias. This is the same split buildEdgeSelect's pre-P-36
// s.cols/s.carried loop made — no .As() on the own-section columns (the
// output name already equals the column name, byte-identical to the old
// inner SELECT's own rendering at both route 2 and route 3, which shared
// the same s.cols loop) — only the carried key(s) get .As(c.as), since
// their source column name differs from their final name. Fable判定
// (2026-09-05): the P-36 issue's own Task 2 code example added .As(c.Name)
// here, which changed TestP05TwoHopComposite_BothCarriedColumnsLandSideBySide's
// pinned SQL substring beyond the derived-table removal this issue scopes
// to; verified via `git show d4afa7e08:.../compose.go` that both routes
// rendered s.cols without AS before P-36, so this keeps that unchanged.
func ownSectionColumns(n flatNode, carried []carriedCol) []any {
	out := make([]any, 0, len(n.edge.childCols()))
	if n.hasKids {
		// Route 3: every column, carried keys included, is already
		// materialized in this node's own CTE under its final name.
		for _, c := range n.edge.childCols() {
			out = append(out, psql.Quote(srcAlias, c.Name))
		}
		return out
	}
	for _, c := range n.edge.childScanCols() {
		out = append(out, psql.Quote(srcAlias, c.Name))
	}
	for _, c := range carried {
		out = append(out, psql.Quote(c.alias, c.column).As(c.as))
	}
	return out
}

// unionBranchOrdColumn (P-36) renders the branch's _sl_ord column.
// Route 3 reads the ordinal its own CTE already materialized; route 2
// computes it here, from the same ORDER BY expressions collapseOrderBy
// used to append at the end of the inner SELECT.
//
// fragment (P-36 gpt-review MEDIUM), when non-nil, is the permittedFragment
// unionBranch already extracted from the edge's options — reused here
// instead of extracting them again, so this ordinal and the WHERE/ORDER BY
// buildEdgeSelect later applies (via edgeSelectSpec.fragment) come from the
// same extraction. nil only for the n.hasKids case, which returns before
// touching fragment at all.
func unionBranchOrdColumn(n flatNode, ordered bool, fragment *permittedFragment) (any, error) {
	if !ordered {
		return psql.Cast(expr.Raw("NULL"), ordDBType).As(colOrd), nil
	}
	if n.hasKids {
		return psql.Quote(srcAlias, ordCol).As(colOrd), nil
	}
	var frag permittedFragment
	if fragment != nil {
		frag = *fragment
	} else {
		var err error
		frag, err = extractOptionsFragment(n.edge.options(), n.edge.relName())
		if err != nil {
			return nil, err
		}
	}
	winMods := make([]bob.Mod[*clause.Window], 0, len(frag.orderBy.Expressions))
	for _, e := range frag.orderBy.Expressions {
		e := e
		winMods = append(winMods, bob.ModFunc[*clause.Window](func(w *clause.Window) {
			w.AppendOrder(e)
		}))
	}
	return psql.F("row_number")(fm.Over(winMods...)).As(colOrd), nil
}

// unionBranchProjection (P-36) lays out the whole island's fixed column
// order for one node section's UNION ALL branch: _sl_set, _sl_ord, root
// columns (NULL-padded), declared count-preload columns (NULL-padded),
// every section's columns (this section's own via ownSectionColumns, every
// other section NULL-padded), and every count section's columns
// (NULL-padded). This is the same layout unionBranch used to build as an
// outer SELECT wrapping buildEdgeSelect's result; it is now handed to
// buildEdgeSelect directly via edgeSelectSpec.projection so the branch
// renders in a single SELECT (P-36 §6/§7 Task 3).
func unionBranchProjection(root partsSpec, plan islandPlan, n flatNode, carried []carriedCol, ordered bool, fragment *permittedFragment) ([]any, error) {
	cols := make([]any, 0, 2+len(root.cols())+len(n.edge.childCols()))
	cols = append(cols, psql.Raw(strconv.Itoa(n.set)).As(colSet))
	ordColExpr, err := unionBranchOrdColumn(n, ordered, fragment)
	if err != nil {
		return nil, err
	}
	cols = append(cols, ordColExpr)
	for _, c := range root.cols() {
		cols = append(cols, psql.Cast(expr.Raw("NULL"), c.DBType).As(c.Name))
	}
	// P-13: root.cols() の NULL padding の直後に、宣言済み count preload も
	// 同様に NULL padding する(この branch は root の行ではないため)。
	for _, dc := range plan.declaredCounts {
		cols = append(cols, psql.Cast(expr.Raw("NULL"), countPreloadDBType).As(dc.Alias))
	}
	for _, m := range plan.sections {
		if m.set == n.set {
			cols = append(cols, ownSectionColumns(n, carried)...)
			continue
		}
		for _, c := range m.edge.childCols() {
			cols = append(cols, psql.Cast(expr.Raw("NULL"), c.DBType).As(c.Name))
		}
	}
	for _, m := range plan.counts {
		for _, col := range m.cols() {
			cols = append(cols, psql.Cast(expr.Raw("NULL"), col.DBType).As(col.Name))
		}
	}
	return cols, nil
}

// unionBranch builds one UNION ALL branch for a node section: its own rows,
// padded with NULLs to the shared column layout, in a single SELECT (P-36).
// The row source, its WHERE match, and the full column layout all come from
// buildEdgeSelect via edgeSelectSpec.projection (issue P-11 §10 step 4,
// P-36 §6).
//
// colOrd is NULL for this section unless the edge's own options (issue
// P-02) requested an ORDER BY: in that case unionBranchOrdColumn computes
// (route 2) or reads back (route 3) a row_number() ordinal instead of
// padding it — for a hasKids node, that column was materialized once on
// the CTE body (route 1) and is read back by name here, so it is never
// computed twice.
//
// hasKidsOrdered (P-36 Codex review Important) is, for a hasKids node
// only, whether that node's CTE body (route 1, built by composeIsland
// BEFORE this function is called for any section) turned out to have an
// ORDER BY — derived from the SAME extractOptionsFragment call that built
// the CTE body itself. Before this parameter existed, route 3 called
// hasOrderBy(n.edge.options()) here, a SECOND, independent extraction of
// the same opts; for a stateful bob.ModFunc (e.g. a mod that only adds
// ORDER BY on every other call) that second extraction could disagree
// with the first, either silently dropping the CTE's real ordinal (this
// branch reads NULL while the CTE has a real _sl_row_ord) or referencing
// a column the CTE never materialized (a runtime SQL error). Ignored for
// a non-hasKids node (route 2 below always derives its own `ordered` from
// its own single extraction).
func unionBranch(root partsSpec, plan islandPlan, n flatNode, hasKidsOrdered bool) (bob.Query, error) {
	child := n.edge.child()
	joins, carried, matchAlias := edgeJoinChain(n.edge)

	spec := edgeSelectSpec{
		edge:       n.edge,
		alias:      srcAlias,
		matchAlias: matchAlias,
	}

	var ordered bool
	var fragment *permittedFragment
	if n.hasKids {
		// Route 3: already filtered by the node's own CTE (built below in
		// composeIsland) — including, for a multi-hop edge, already
		// JOINed through its intermediate table(s) there. No matching, no
		// JOINs, and no opts to redo here: the CTE already materialized
		// the carried key(s) under their final _sl_k<i> names, so
		// ownSectionColumns (via n.edge.childCols()) reads them back by
		// name like any other column — spec.joins/spec.carried stay their
		// zero value (joins/carried computed above are simply unused on
		// this path). No opts of its own, so no fragment to extract —
		// fragment stays nil and unionBranchOrdColumn's n.hasKids branch
		// never touches it.
		spec.from = psql.Quote(cteName(n.set))
		ordered = hasKidsOrdered
	} else {
		// Route 2: read the table directly, semi-joined against the
		// parent CTE, and — for a multi-hop edge — JOINed through its
		// intermediate table(s) (edgeJoinChain above). This is the only
		// physical SELECT for this edge, so its own opts, joins, and
		// carried-key SELECT all apply here.
		//
		// extractOptionsFragment is called exactly once here (P-36
		// gpt-review MEDIUM): the resulting fragment decides `ordered`,
		// feeds unionBranchOrdColumn's _sl_ord ordinal below, and is
		// handed to buildEdgeSelect via spec.fragment, so the ordinal and
		// the WHERE/ORDER BY actually applied to the branch always agree
		// — even when opts.mods carries a stateful bob.ModFunc that would
		// otherwise render a different fragment on a second extraction.
		opts := n.edge.options()
		if len(opts.mods) > 0 {
			frag, err := extractOptionsFragment(opts, n.edge.relName())
			if err != nil {
				return nil, err
			}
			// P-37 Critical (gpt-review): unionBranchOrdColumn (below, via
			// unionBranchProjection) reads frag.orderBy.Expressions to build
			// _sl_ord's row_number() OVER (ORDER BY ...) BEFORE
			// buildEdgeSelect ever sees spec.fragment — so a qualified
			// column reference inside that ORDER BY must be rewritten here,
			// not left for buildEdgeSelect's own rewriteQualifiedRefs call
			// (compose.go's buildEdgeSelect) to catch, or it survives
			// unrewritten inside _sl_ord for every hasKids=false, ordered
			// branch.
			frag = rewriteQualifiedRefs(frag, sectionAliasMap(n.edge))
			fragment = &frag
			ordered = len(frag.orderBy.Expressions) > 0
		}

		spec.from = tableExpr(child.schemaName(), child.tableName())
		spec.semiFromCTE = cteName(n.parentSet)
		spec.opts = opts
		spec.collapseOrderBy = true
		spec.joins = joins
		spec.carried = carried
		spec.fragment = fragment
	}

	projection, err := unionBranchProjection(root, plan, n, carried, ordered, fragment)
	if err != nil {
		return nil, err
	}
	spec.projection = projection

	inner, err := buildEdgeSelect(spec)
	if err != nil {
		return nil, err
	}
	return inner, nil
}

// countBranch builds one UNION ALL branch for a COUNT section: its own
// aggregate rows (join key + count(*), grouped by the key, semi-joined
// against the parent CTE — the same route 2 a leaf row section takes),
// padded with NULLs to the shared column layout.
//
// A count section is always a leaf, so there is no CTE read-back route
// here: this is the one and only physical SELECT for the edge, and its
// caller options apply exactly once, right here.
//
// colOrd is always NULL for a count branch: count rows are consumed
// through a key→count map (stitchCounts), so no row order ever needs to
// survive the UNION ALL, and collapseOrderBy (a row-section mechanism)
// stays false. The edge's own ORDER BY, if any, is left verbatim inside
// the aggregate subquery below — alone it cannot change any count, and
// combined with LIMIT/OFFSET it decides which key groups are counted,
// exactly like the same mods appended to bob's own LoadCount<Rel>.
func countBranch(root partsSpec, plan islandPlan, c countPlan) (bob.Query, error) {
	child := c.edge.child()
	// R-24: for a multi-hop count, joins/matchAlias let buildEdgeSelect
	// JOIN through the intermediate table(s) and quote SELECT/GROUP BY
	// against the FIRST intermediate table's alias instead of the FINAL
	// table's — see buildEdgeSelect's own colAlias comment (§6.4 of
	// P-34-multihop-count.md). carried is discarded: a count section never
	// adds a SQL carried-key column (§6.2 of the same doc — the SELECT
	// stays "key columns + count(*)"). For a single-hop count,
	// edgeJoinChain returns (nil, nil, srcAlias), so joins is a no-op and
	// matchAlias equals srcAlias — byte-identical SQL to before R-24.
	joins, _, matchAlias := edgeJoinChain(c.edge)
	inner, err := buildEdgeSelect(edgeSelectSpec{
		edge:        c.edge,
		cols:        c.keyCols,
		from:        tableExpr(child.schemaName(), child.tableName()),
		alias:       srcAlias,
		matchAlias:  matchAlias,
		joins:       joins,
		semiFromCTE: cteName(c.parentSet),
		opts:        c.edge.options(),
		countAs:     c.countCol.Name,
		groupBy:     c.keyCols,
	})
	if err != nil {
		return nil, err
	}

	cols := make([]any, 0, 2+len(root.cols()))
	cols = append(cols, psql.Raw(strconv.Itoa(c.set)).As(colSet))
	cols = append(cols, psql.Cast(expr.Raw("NULL"), ordDBType).As(colOrd))
	for _, col := range root.cols() {
		cols = append(cols, psql.Cast(expr.Raw("NULL"), col.DBType).As(col.Name))
	}
	// P-13: root.cols() の NULL padding の直後に、宣言済み count preload も
	// 同様に NULL padding する(この branch も root の行ではないため)。
	for _, dc := range plan.declaredCounts {
		cols = append(cols, psql.Cast(expr.Raw("NULL"), countPreloadDBType).As(dc.Alias))
	}
	for _, n := range plan.sections {
		for _, col := range n.edge.childCols() {
			cols = append(cols, psql.Cast(expr.Raw("NULL"), col.DBType).As(col.Name))
		}
	}
	for _, m := range plan.counts {
		for _, col := range m.cols() {
			if m.set == c.set {
				cols = append(cols, psql.Quote(srcAlias, col.Name).As(col.Name))
				continue
			}
			cols = append(cols, psql.Cast(expr.Raw("NULL"), col.DBType).As(col.Name))
		}
	}

	return psql.Select(
		sm.Columns(cols...),
		sm.From(inner).As(srcAlias),
	), nil
}

// composeIsland builds the sectioned UNION ALL query for one island.
// The parent query is embedded unmodified inside the _sl_src CTE.
//
// Column layout (fixed): _sl_set, _sl_ord, <root cols>, <declared count cols
// (plan.declaredCounts)>, <node cols in set order>, <count key cols +
// _sl_cnt<set>, in count set order>.
//
// parentOrdCol names the column _sl_src's own row_number() should ORDER BY,
// when the parent query has one; empty means _sl_ord falls back to
// row_number() OVER () with no ORDER BY, leaning on the query planner
// preserving materialization order without an SQL-level guarantee (issue
// P-02 §10.2/A-02-1's own doc comment already accepted this for the
// top-level root). Two callers supply a non-empty parentOrdCol:
//
//   - runThen's route 4: sectionload builds the parent query itself via
//     buildEdgeSelect(collapseOrderBy: true), so when the edge requested an
//     ORDER BY, that query already carries an explicit, real ordinal column
//     of its own (ordCol) (issue P-02 Important-2, Codex review 2026-08-09).
//   - All/Build, for a caller-supplied top-level parentQuery that itself has
//     an ORDER BY: installParentOrdColumn (run.go) appends a row_number()
//     window ordered by that same ORDER BY onto the parent query, so
//     _sl_src's row_number() can follow it explicitly instead of the same
//     unguaranteed pass-through above (Copilot review, PR #1997,
//     compose.go:704, issue P-02 addendum).
//
// plan.parentOuterOrd/parentOuterAlias (run.go's outerParentOrd) is a third,
// preferred source for the same ordinality: when set, _sl_src's
// row_number() OVER (...) is placed OUTSIDE the parent subquery instead of
// relying on parentOrdCol's inside placement — see outerParentOrd's doc
// comment for why a parent LIMIT needs this to keep using a top-N sort.
func composeIsland(parent bob.Query, root partsSpec, plan islandPlan, maxColumns int, parentOrdCol string) (bob.Query, error) {
	if err := checkDeclaredCountCollisions(root, plan); err != nil {
		return nil, err
	}
	if err := checkWidth(root, plan, maxColumns); err != nil {
		return nil, err
	}

	// _sl_src: wrap the parent query and add the ordinality column.
	//
	// parentSource (parentexpr.go) builds the parent exactly once here,
	// checking it for reserved identifiers at the same time — instead of
	// All/Build building it a second time just for that check.
	// parentWrapAlias is the FROM-alias of the parent subquery inside
	// _sl_src's own wrapper SELECT — NOT srcAlias (const above, "_sl_t",
	// each section branch's OWN row source alias further down this
	// function). Naming it srcAlias here would shadow that package
	// constant for the rest of composeIsland's body and silently break
	// every "_sl_t"-aliased reference below (caught by
	// TestP05Baseline/nested going from byte-identical to
	// "q"-referencing during this function's P-35 addition).
	var srcCols []any
	var ordExpr any
	var parentWrapAlias string
	switch {
	case len(plan.parentOuterOrd) > 0:
		// Outer placement (run.go's outerParentOrd): the wrapper's alias
		// is the parent table's own name, matching what
		// plan.parentOuterOrd's expressions already reference ("<table>".
		// "<column>"), so they can be reused verbatim as the outer
		// row_number()'s ORDER BY.
		parentWrapAlias = plan.parentOuterAlias
		srcCols = []any{psql.Raw(quoteIdent(parentWrapAlias) + ".*")}
		winMods := make([]bob.Mod[*clause.Window], 0, len(plan.parentOuterOrd))
		for _, e := range plan.parentOuterOrd {
			winMods = append(winMods, bob.ModFunc[*clause.Window](func(w *clause.Window) {
				w.AppendOrder(e)
			}))
		}
		ordExpr = psql.F("row_number")(fm.Over(winMods...)).As(colOrd)
	case parentOrdCol == "":
		parentWrapAlias = "q"
		srcCols = []any{psql.Raw("q.*")}
		ordExpr = psql.Raw("row_number() OVER () AS " + quoteIdent(colOrd))
	default:
		// q.* would also expose parentOrdCol itself (it is a real column
		// on parent's own SELECT list, not a window-only value) — list the
		// known columns explicitly instead, so the fixed column layout
		// this whole function promises stays exactly root.cols() plus
		// plan.declaredCounts (the same layout checkWidth and
		// expectedColumns count), nothing more.
		parentWrapAlias = "q"
		cols := root.cols()
		srcCols = make([]any, 0, len(cols)+len(plan.declaredCounts))
		for _, c := range cols {
			srcCols = append(srcCols, psql.Quote("q", c.Name))
		}
		// P-41: the declared PreloadCount columns are on the parent's own
		// SELECT list too, and section0List reads them back by name off
		// _sl_src — the other two branches pass them through with .*, so
		// this explicit list must carry them as well, in the same order
		// section0List uses (root.cols() first, declaredCounts after).
		for _, dc := range plan.declaredCounts {
			srcCols = append(srcCols, psql.Quote("q", dc.Alias))
		}
		ordExpr = psql.Raw("row_number() OVER (ORDER BY " + quoteIdent("q") + "." + quoteIdent(parentOrdCol) + ") AS " + quoteIdent(colOrd))
	}
	wrapped := psql.Select(
		sm.Columns(append(srcCols, ordExpr)...),
		sm.From(parentSource{parent}).As(parentWrapAlias),
	)

	mods := []bob.Mod[*pdialect.SelectQuery]{
		sm.With(cteSrc).As(wrapped).Materialized(),
	}

	// CTEs for nodes that have children (they are referenced twice:
	// once to emit their section, once as the semi-join key source).
	// Route 1 of buildEdgeSelect (issue P-11 §11.3). The edge's own
	// options (issue P-02) apply here — its CTE body is the one physical
	// SELECT for this edge — so both the section's own rows (read back by
	// unionBranch, route 3) and the semi-join population for its own
	// children/grandchildren see the same filtered/limited/ordered set,
	// matching bob's nested-ThenLoad LIMIT semantics (issue P-02 A-02-2).
	//
	// hasKidsOrdered (P-36 Codex review Important) records, for every
	// hasKids node, whether its opts turned out to have an ORDER BY —
	// derived from the SAME extractOptionsFragment call below that builds
	// this CTE body, and handed to unionBranch's route 3 later in this
	// function instead of letting it re-extract the same opts a second
	// time (see unionBranch's hasKidsOrdered doc for what could go wrong
	// with two independent extractions of a stateful opts mod).
	hasKidsOrdered := make(map[int]bool, len(plan.sections))
	for _, n := range plan.sections {
		if !n.hasKids {
			continue
		}
		child := n.edge.child()
		joins, carried, matchAlias := edgeJoinChain(n.edge)
		opts := n.edge.options()
		var fragment *permittedFragment
		if len(opts.mods) > 0 {
			frag, err := extractOptionsFragment(opts, n.edge.relName())
			if err != nil {
				return nil, err
			}
			fragment = &frag
			hasKidsOrdered[n.set] = len(frag.orderBy.Expressions) > 0
		}
		cteBody, err := buildEdgeSelect(edgeSelectSpec{
			edge: n.edge,
			// childScanCols(), not childCols(): the carried key(s), for a
			// multi-hop edge, come from the intermediate-table JOIN below
			// (spec.carried), not from `child` itself (P-05).
			cols:            n.edge.childScanCols(),
			from:            tableExpr(child.schemaName(), child.tableName()),
			alias:           srcAlias,
			joins:           joins,
			carried:         carried,
			matchAlias:      matchAlias,
			semiFromCTE:     cteName(n.parentSet),
			opts:            opts,
			collapseOrderBy: true,
			fragment:        fragment,
		})
		if err != nil {
			return nil, err
		}
		mods = append(mods, sm.With(cteName(n.set)).As(cteBody).Materialized())
	}

	// Leading branch: section 0 (the parent).
	mods = append(
		mods,
		// expr.Raw, not psql.Raw (issue P-29, the same failure class as
		// P-09's matchWhere/parentKeySubquery fix): section0List joins
		// column names and NULL-cast DB type names verbatim, and a
		// PostgreSQL quoted identifier may legally contain "?" (e.g.
		// Column{Name: "flag?"}). psql.Raw/expr.Clause's "?" scanning would
		// miscount it as a bob argument placeholder; expr.Raw writes the
		// string as-is with no scanning at all.
		sm.Columns(expr.Raw(section0List(root, plan))),
		sm.From(psql.Raw(quoteIdent(cteSrc))),
	)

	// One UNION ALL branch per node section (routes 2/3 of buildEdgeSelect,
	// via unionBranch).
	for _, n := range plan.sections {
		branch, err := unionBranch(root, plan, n, hasKidsOrdered[n.set])
		if err != nil {
			return nil, err
		}
		mods = append(mods, sm.UnionAll(branch))
	}

	// One more UNION ALL branch per count section (issue P-04 A-04-1:
	// counts ride the SAME island, never a separate query).
	for _, c := range plan.counts {
		branch, err := countBranch(root, plan, c)
		if err != nil {
			return nil, err
		}
		mods = append(mods, sm.UnionAll(branch))
	}

	return psql.Select(mods...), nil
}

// checkDeclaredCountCollisions (P-13、W-4) は宣言 alias が root 列・section
// 列・_sl_cnt<k> 列のいずれとも衝突しないか、または宣言同士で重複していない
// か(M-5)を検査する。plan.declaredCounts 自身を含めない基準集合が要るので、
// 一時的に declaredCounts を空にしたコピー(islandPlan は値型なので plan
// 自体は書き換わらない)で expectedColumns を呼ぶ。[2:] で先頭の
// _sl_set/_sl_ord を除く(これらは quoteIdent された予約名であり利用者の
// 宣言と衝突する経路が無いので比較対象に含めない)。
func checkDeclaredCountCollisions(root partsSpec, plan islandPlan) error {
	base := plan
	base.declaredCounts = nil
	existing := make(map[string]bool, len(base.sections)+len(base.counts))
	for _, name := range expectedColumns(root, base)[2:] {
		existing[name] = true
	}
	// M-5: 宣言リスト自体の alias 重複も検査する。同じ preloader を2回
	// 付けると _sl_src の出力列に同名が2つ並び、"_sl_src"."__count_X" が
	// PostgreSQL の ambiguous column reference エラーになる(推測: 実測は
	// していない。_sl_src の内側は、外側方式と ORDER BY 無しの分岐では
	// "<親テーブル>".* / q.* が親クエリの全列を露出し、内側方式の分岐では
	// 明示列挙が同じ alias を2回並べるため、いずれの経路でも同名列が2列
	// できてこの参照が曖昧になる)。checkDeclaredCountCollisions はこの
	// 経路も同じエラー型で拒否する。
	seen := make(map[string]bool, len(plan.declaredCounts))
	for _, dc := range plan.declaredCounts {
		if existing[dc.Alias] || seen[dc.Alias] {
			return &DeclaredCountColumnConflictError{Alias: dc.Alias}
		}
		seen[dc.Alias] = true
	}
	return nil
}

func checkWidth(root partsSpec, plan islandPlan, maxColumns int) error {
	total := 2 + len(root.cols()) + len(plan.declaredCounts)
	if total > maxColumns {
		return &TooWideError{Total: total, Max: maxColumns, AtRel: "(parent)"}
	}
	for _, n := range plan.sections {
		total += len(n.edge.childCols())
		if total > maxColumns {
			return &TooWideError{Total: total, Max: maxColumns, AtRel: n.edge.relName()}
		}
	}
	for _, c := range plan.counts {
		total += len(c.cols())
		if total > maxColumns {
			return &TooWideError{Total: total, Max: maxColumns, AtRel: c.edge.relName()}
		}
	}
	return nil
}

// section0List builds the SELECT list of the leading (parent) branch:
// 0, _sl_ord, all src columns, NULL padding for every node column and
// every count column. UNION ALL takes the result column NAMES from this
// first branch, so every padded column carries an explicit alias.
func section0List(root partsSpec, plan islandPlan) string {
	parts := make([]string, 0, 8)
	parts = append(parts, "0 AS "+colSet)
	parts = append(parts, quoteIdent(cteSrc)+"."+quoteIdent(colOrd))
	for _, c := range root.cols() {
		parts = append(parts, quoteIdent(cteSrc)+"."+quoteIdent(c.Name))
	}
	// P-13: 宣言済み count preload は root.cols() に含まれない、親branch
	// 専用の追加列。_sl_src の内側には既に存在する(PreloadCount が足した
	// 相関サブクエリ列)ので、ここで名前指定して読み出すだけでよい。
	for _, dc := range plan.declaredCounts {
		parts = append(parts, quoteIdent(cteSrc)+"."+quoteIdent(dc.Alias))
	}
	for _, n := range plan.sections {
		for _, c := range n.edge.childCols() {
			parts = append(parts, "NULL::"+c.DBType+" AS "+quoteIdent(c.Name))
		}
	}
	for _, cp := range plan.counts {
		for _, c := range cp.cols() {
			parts = append(parts, "NULL::"+c.DBType+" AS "+quoteIdent(c.Name))
		}
	}
	return strings.Join(parts, ", ")
}
