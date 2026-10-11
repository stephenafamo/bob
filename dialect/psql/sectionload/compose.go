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
	// instead of reading a real ordinal: row_number() OVER () (what ordered
	// branches actually read back) returns bigint in PostgreSQL. A count
	// branch's NULL padding could reuse countDBType (also "bigint") instead
	// of this — harmless today, but the two types have no reason to stay
	// equal, so colOrd gets its own constant.
	ordDBType = "bigint"
)

// countPreloadDBType is __count_<name> columns' own SQL type wherever a
// branch NULL-pads them: bob's countPreloader always wraps count(*) in a
// correlated subquery (bob_counts.bob.go.tpl), which PostgreSQL returns
// as bigint — the same value countDBType (node.go, for count sections)
// happens to hold today, but the two have no reason to stay equal, so
// this gets its own constant (same reasoning as ordDBType above).
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

// tableExpr renders a table name as a bob expression, reproducing bob's
// own NameExpr priority (dialect/psql/view.go) exactly: a non-empty
// generated schema is always fixed and never overridden by a runtime
// psql.UseSchema(ctx, ...) context value, matching how bob's own
// generated views behave. A generated schema of "" (the common case)
// delegates to orm.SchemaTable, which reads orm.CtxUseSchema from the
// query-build context at WriteSQL time — the same mechanism the parent
// query already uses, so the parent and every relation sectionload
// composes agree on which schema they read from.
func tableExpr(schema, table string) bob.Expression {
	if schema == "" {
		return orm.SchemaTable(table)
	}
	return psql.Quote(schema, table)
}

// hopAliasPrefix/hopAlias name the intermediate table(s) a multi-hop edge
// JOINs through. Scoped to one edge's own subquery/CTE (never the outer
// island layout), so — unlike _sl_set/_sl_ord/_sl_src/_sl_n<k>/_sl_cnt<k>
// — these never need reservedNameRe protection.
const hopAliasPrefix = "_sl_h"

func hopAlias(i int) string { return fmt.Sprintf("%s%d", hopAliasPrefix, i) }

// sideToAlias returns the alias Sides[i].To is read under: the final
// table's own alias (srcAlias) when i is the last side, otherwise the
// NEXT hop's alias (that hop's From IS this side's To — bob's own
// Relationship.Validate() guarantees Sides[i].To == Sides[i+1].From, see
// parts.go's Side doc). For a single-hop edge (n==1), sideToAlias(0, 1)
// is srcAlias — the alias every single-hop call site uses, which is
// exactly why matchAlias needs no separate single-hop branch.
func sideToAlias(i, n int) string {
	if i == n-1 {
		return srcAlias
	}
	return hopAlias(i + 1)
}

// sectionAliasMap maps each table name this section's own SELECT reads to
// the alias it is read under inside that SELECT. A caller mod that
// qualifies a column with the table's own name (which every generated
// <Table>.Columns.<Col> expression does) can then be rewritten to the
// alias the FROM clause actually declares.
//
// Single-hop: e.child()'s own table -> srcAlias.
// Multi-hop: e.sides()[i].ToTable -> sideToAlias(i, len(sides)) for every
// i, which is srcAlias for the last side and hopAlias(i+1) otherwise.
// Route 3 (the section re-reads its own CTE) gets the same child-table
// entry: the CTE exposes the child's columns under their own names, so
// "<child>"."<col>" -> "_sl_t"."<col>" is still the right rewrite.
//
// e.sides()[i].ToTable is "" for a single-hop Edge/CountEdge that never
// set Sides (sides() synthesizes a length-1 Side from the flat
// FromCols/ToCols fields, which carry no table names) — that entry is
// skipped, leaving only the child-table entry, which is exactly what a
// single-hop section needs.
//
// A table name recorded at two DIFFERENT aliases (a self-referencing
// multi-hop edge that reads the same table under two different hop
// aliases) is ambiguous — a bare table-name qualifier cannot say which
// occurrence it means — so that name is removed from the map entirely: a
// caller mod that qualifies with it is left unresolved and still fails
// loudly, rather than silently resolving to whichever occurrence happened
// to be recorded last. The same name recorded twice at the SAME alias
// (e.g. the child table via both the explicit entry above and the last
// side's ToTable) is not ambiguous and is kept.
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
// "<alias>"."<column>" AS "<as>".
type carriedCol struct {
	alias  string
	column string
	as     string
}

// hopChainSpec is the minimal interface edgeJoinChain actually needs:
// sides() and carriedCols(). edgeSpec (row edges) and countEdgeSpec
// (count edges) both satisfy it structurally, so the same JOIN-chain
// builder serves row edges and multi-hop counts alike (SQL INNER JOIN is
// commutative, so FROM-last-table-backward-JOIN produces the same result
// set as bob's own FROM-first-table-forward-JOIN LoadCount<Rel> shape).
type hopChainSpec interface {
	sides() []Side
	carriedCols() []Column
}

// edgeJoinChain builds the JOIN chain, carried-key SELECT list, and match
// alias for one edge's PHYSICAL SELECT — routes 1, 2, and 4 of
// buildEdgeSelect (route 3 re-reads an already-joined CTE by name and
// needs none of this; see unionBranch). For a single-hop edge
// (len(sides())<=1) this returns (nil, nil, srcAlias): no JOINs, no
// carried columns, and the match alias every single-hop route uses — so
// buildEdgeSelect renders the same SQL as it would without this function.
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
func edgeJoinChain(e hopChainSpec) ([]hopJoin, []carriedCol, string) {
	sides := e.sides()
	n := len(sides)
	matchAlias := sideToAlias(0, n)
	if n <= 1 {
		return nil, nil, matchAlias
	}

	joins := make([]hopJoin, 0, n-1)
	for i := n - 1; i >= 1; i-- {
		s := sides[i]
		fromAlias := hopAlias(i)
		toAlias := sideToAlias(i, n)
		on := make([]bob.Expression, 0, len(s.ToCols)+len(s.Where))
		for j, toCol := range s.ToCols {
			on = append(on, psql.Quote(toAlias, toCol).EQ(psql.Quote(fromAlias, s.FromCols[j])))
		}
		// This hop's OWN fixed condition(s) (Sides[i].Where, i>=1): unlike side
		// 0's, these never reach matchWhere/parentKeySubquery (those only see
		// edgeWhere()==sides()[0].Where) — they belong in this JOIN's own ON
		// clause, on whichever side of the hop they name.
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
	carried := make([]carriedCol, 0, len(side0.ToCols))
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
// same builder:
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
	cols  []Column       // columns to SELECT, in Columns order (may be restricted by a column plan)
	from  bob.Expression // table or CTE name expression to read from
	alias string         // alias assigned to `from` inside this SELECT

	// joins/carried/matchAlias: a multi-hop edge's JOIN chain through its
	// intermediate table(s), the carried-key SELECT list, and the alias the
	// semi-join/keys match and side-0 "to" fixed condition are read against.
	// All three come from edgeJoinChain, built by the caller BEFORE this spec
	// (edgeJoinChain needs edgeSpec, a strictly larger interface than the
	// edgeCore this struct embeds). Set at routes 1, 2, and 4
	// (buildEdgeSelect's three PHYSICAL routes); left empty/"" at route 3 (a
	// with-children node's own CTE read-back), which reads an already-joined
	// row back by name and needs no JOIN of its own — see unionBranch.
	// joins/carried are nil and matchAlias == srcAlias for every single-hop
	// edge, at every route, so single-hop SQL is unaffected.
	joins      []hopJoin
	carried    []carriedCol
	matchAlias string

	// Matching against the parent. At most one is set; neither set means
	// `from` needs no matching of its own (route 3 above).
	semiFromCTE string  // same-island: name of the parent CTE to semi-join against
	keys        [][]any // Then boundary: parent key tuples, compared by row-value IN

	// opts carries the edge's own caller-provided SELECT mods.
	// Left zero at route 3 (a with-children node's UNION ALL branch, read
	// back from its own CTE): that CTE's body (route 1) already applied
	// opts once; applying again here would run WHERE/LIMIT/OFFSET twice.
	opts SelectOptions
	// collapseOrderBy is set at routes 1, 2, and 4 (this SELECT becomes one
	// arm of a UNION ALL, the body of a CTE read back by one, or a
	// Then(...) boundary's own parent query): a literal ORDER BY inside a
	// UNION ALL arm does not survive the union, and a route 4 query becomes
	// the new island's own root, which composeIsland wraps in its own
	// order-restoring row_number() OVER () (_sl_src) the same way every plain
	// root query is — so both need the same rewrite buildEdgeSelect applies
	// here: any ORDER BY opts add is rewritten into a row_number() OVER
	// (ORDER BY ...) ordinal column instead, and the literal ORDER BY is
	// cleared. Left false only at route 3 (a with-children node's UNION ALL
	// branch, read back from its own CTE): that CTE's body (route 1) already
	// applied opts once, including this same collapse, so route 3 must not
	// repeat it.
	collapseOrderBy bool

	// countAs, when non-empty, turns this SELECT into a count section's
	// aggregate: count(*) is appended to the select list under that alias
	// (the layout's _sl_cnt<set> column), and groupBy names the columns to
	// GROUP BY — always the edge's own child-side join key, so the result is
	// exactly one row per parent key. Empty at every row-section route.
	countAs string
	groupBy []Column

	// projection, when non-empty, is this SELECT's column list
	// verbatim — the caller (unionBranch) has already laid out the whole
	// island's column order, including the NULL padding for every column
	// this branch does not own. cols/carried/countAs are then ignored.
	// Empty at routes 1 and 4 (a CTE body and a Then boundary's own parent
	// query are not UNION ALL arms, so they have no island layout to
	// follow).
	projection []any

	// fragment, when non-nil, is the permittedFragment unionBranch already
	// extracted from s.opts via extractOptionsFragment — the SAME extraction
	// unionBranchOrdColumn used to compute the branch's _sl_ord row_number()
	// ordinal. When set, buildEdgeSelect reuses it instead of extracting
	// s.opts a second time, so the WHERE/ORDER BY actually applied to this
	// SELECT and the ordinal column unionBranchOrdColumn already emitted are
	// guaranteed to come from one and the same SelectOptions evaluation, even
	// when opts.mods carries a stateful bob.ModFunc (e.g. a counter) that
	// would otherwise render a different fragment on a second call. Left nil
	// at routes 1, 3, and 4 (route 3 passes no opts at all; routes 1 and 4
	// have no projection/ordinal of their own to keep in sync with), where
	// buildEdgeSelect falls back to extracting s.opts itself.
	fragment *permittedFragment
}

// ordCol is the name of the per-section row-ordinal column buildEdgeSelect
// adds to its own SELECT list when collapsing an ORDER BY opt. It is
// scoped to one edge's own subquery/CTE and never appears in the
// generated column contract, so it does not collide with _sl_ord (the
// root-restoring ordinal composeIsland already owns) or with any table
// column name.
const ordCol = "_sl_row_ord"

// sortColPrefix names the columns outerParentOrd hoists onto the parent
// query's SELECT list when an ORDER BY expression cannot be referenced
// from the outer wrapper's single-table alias (a joined table's column,
// an EXISTS subquery, any computed expression). Like ordCol above, this
// name deliberately sits OUTSIDE reservedNameRe's namespace
// (parentexpr.go): the check exists to catch a CALLER's column colliding
// with sectionload's internal names, and these columns are sectionload's
// own additions to the parent's SELECT list, not the caller's. The i-th
// hoisted key is sortColPrefix + strconv.Itoa(i).
const sortColPrefix = "_sl_row_sort"

// buildEdgeSelect builds "SELECT <cols> FROM <from> AS <alias> [WHERE
// <match>]" for one edge's own rows, then merges in s.opts on top. It
// never returns a partial result: the WHERE/JOIN structure lives entirely
// inside this function and matchWhere, so no caller needs to concatenate
// SQL text.
//
// s.opts never touches this query directly — extractOptionsFragment (see
// options.go) applies it to an isolated, empty query instead, and only the
// permitted fragment (WHERE/ORDER BY/LIMIT/OFFSET/GROUP BY/HAVING) is
// merged in via permittedFragment.applyTo, after the join condition above.
// A caller mod therefore has no way to see, read, or delete the match
// condition sectionload itself depends on — not merely "applied, then
// checked for a diff" (an apply-then-diff design would never compare
// Where, so a mod that cleared it would pass unnoticed).
func buildEdgeSelect(s edgeSelectSpec) (bob.BaseQuery[*pdialect.SelectQuery], error) {
	// A count section's own s.cols/s.groupBy (c.keyCols) name a column of
	// s.alias (the FINAL child table) only for a single-hop count. For a
	// multi-hop count, they name a column of the FIRST INTERMEDIATE table
	// instead — the same table edgeJoinChain's matchAlias already points at —
	// so SELECT/GROUP BY must be quoted against matchAlias there, not
	// s.alias. A row section never sets countAs, so colAlias stays s.alias
	// (unchanged). A single-hop count's own matchAlias equals srcAlias
	// (==s.alias here), so this branch renders the same SQL as before for
	// single-hop counts.
	colAlias := s.alias
	if s.countAs != "" && s.matchAlias != "" {
		colAlias = s.matchAlias
	}

	var cols []any
	if len(s.projection) > 0 {
		// The caller (unionBranch) has already laid out the whole
		// island's column order, NULL padding included — use it verbatim
		// instead of building the column list from s.cols/s.carried/
		// s.countAs.
		cols = s.projection
	} else {
		cols = make([]any, 0, len(s.cols)+len(s.carried)+1)
		for _, c := range s.cols {
			cols = append(cols, psql.Quote(colAlias, c.Name))
		}
		// s.carried: a multi-hop edge's carried join-key column(s),
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
	// s.joins: a multi-hop edge's intermediate-table JOIN chain,
	// in the order edgeJoinChain built it (last side first). Empty for a
	// single-hop edge, so this loop is a no-op.
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
		// unionBranch already extracted this fragment once (to compute the
		// branch's _sl_ord ordinal via unionBranchOrdColumn) — reuse it instead
		// of extracting s.opts a second time, so both read the same WHERE/ORDER BY
		// even when opts.mods carries a stateful bob.ModFunc.
		frag = *s.fragment
	} else {
		var err error
		frag, err = extractOptionsFragment(s.opts, s.edge.relName())
		if err != nil {
			return bob.BaseQuery[*pdialect.SelectQuery]{}, err
		}
	}
	// Resolve a caller mod's qualified column references (e.g. a generated
	// <Table>.Columns.<Col>, which always renders "<table>"."<col>") against
	// the alias this section's own FROM clause actually declares, before
	// merging the fragment in.
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
		// On the projection path, unionBranchOrdColumn has already placed the
		// same row_number() expression at the _sl_ord position. Here the ORDER BY
		// only needs to be cleared.
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
// parentKeySubquery, matching the row-value-IN condition exactly.
//
// The match columns (toCols()==sides()[0].ToCols) and side-0's own "to"
// fixed condition are read from s.matchAlias, not s.alias: for a
// multi-hop edge, sides()[0].To is the FIRST intermediate table, which
// edgeJoinChain aliases as hopAlias(1) — never s.alias (srcAlias, the
// FINAL table). s.matchAlias falls back to s.alias when empty. Every
// row-section route (1/2/3/4) sets matchAlias explicitly, and
// countBranch's call also sets matchAlias (via edgeJoinChain, same as a
// row section), so this fallback is a defensive default only, not a
// load-bearing path for any current caller.
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
// writes it verbatim without scanning it for "?" placeholders: a
// relationship condition like `data ? 'active'` (a jsonb operator) must not
// be miscounted as a bob argument placeholder the way psql.Raw/expr.Clause's
// textual "?" scanning would. The miscount would fail loudly at bob.Build
// time (an args/placeholder-count mismatch, expr/raw.go's rawError) rather
// than silently producing a wrong query, but a legal relationship
// configuration should not build-fail at all.
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

// ownSectionColumns renders the columns this branch actually owns, in
// childCols() order: the child table's own columns read from srcAlias,
// then a multi-hop edge's carried key column(s) read from the intermediate
// table's own alias. There is no .As() on the own-section columns (the
// output name already equals the column name, and both route 2 and route 3
// share the same rendering) — only the carried key(s) get .As(c.as), since
// their source column name differs from their final name.
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

// unionBranchOrdColumn renders the branch's _sl_ord column.
// Route 3 reads the ordinal its own CTE already materialized; route 2
// computes it here, from the same ORDER BY expressions collapseOrderBy
// used to append at the end of the inner SELECT.
//
// fragment, when non-nil, is the permittedFragment unionBranch already
// extracted from the edge's options — reused here instead of extracting
// them again, so this ordinal and the WHERE/ORDER BY buildEdgeSelect later
// applies (via edgeSelectSpec.fragment) come from the same extraction. nil
// only for the n.hasKids case, which returns before touching fragment at
// all.
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

// unionBranchProjection lays out the whole island's fixed column order for
// one node section's UNION ALL branch: _sl_set, _sl_ord, root columns
// (NULL-padded), declared count-preload columns (NULL-padded), every
// section's columns (this section's own via ownSectionColumns, every other
// section NULL-padded), and every count section's columns (NULL-padded).
// It is handed to buildEdgeSelect via edgeSelectSpec.projection so the
// branch renders in a single SELECT.
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
	// Right after root.cols()' NULL padding, NULL-pad the declared count
	// preload columns too (this branch is not a root row).
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
// padded with NULLs to the shared column layout, in a single SELECT.
// The row source, its WHERE match, and the full column layout all come from
// buildEdgeSelect via edgeSelectSpec.projection.
//
// colOrd is NULL for this section unless the edge's own options requested
// an ORDER BY: in that case unionBranchOrdColumn computes (route 2) or
// reads back (route 3) a row_number() ordinal instead of padding it — for
// a hasKids node, that column was materialized once on the CTE body
// (route 1) and is read back by name here, so it is never computed twice.
//
// hasKidsOrdered is, for a hasKids node only, whether that node's CTE body
// (route 1, built by composeIsland BEFORE this function is called for any
// section) turned out to have an ORDER BY — derived from the SAME
// extractOptionsFragment call that built the CTE body itself. Calling
// hasOrderBy(n.edge.options()) here instead would be a SECOND,
// independent extraction of the same opts; for a stateful bob.ModFunc
// (e.g. a mod that only adds ORDER BY on every other call) that second
// extraction could disagree with the first, either silently dropping the
// CTE's real ordinal (this branch reads NULL while the CTE has a real
// _sl_row_ord) or referencing a column the CTE never materialized (a
// runtime SQL error). Ignored for a non-hasKids node (route 2 below always
// derives its own `ordered` from its own single extraction).
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
		// extractOptionsFragment is called exactly once here: the resulting
		// fragment decides `ordered`, feeds unionBranchOrdColumn's _sl_ord
		// ordinal below, and is handed to buildEdgeSelect via spec.fragment, so
		// the ordinal and the WHERE/ORDER BY actually applied to the branch always
		// agree — even when opts.mods carries a stateful bob.ModFunc that would
		// otherwise render a different fragment on a second extraction.
		opts := n.edge.options()
		if len(opts.mods) > 0 {
			frag, err := extractOptionsFragment(opts, n.edge.relName())
			if err != nil {
				return nil, err
			}
			// unionBranchOrdColumn (below, via unionBranchProjection) reads
			// frag.orderBy.Expressions to build _sl_ord's row_number() OVER (ORDER BY
			// ...) BEFORE buildEdgeSelect ever sees spec.fragment — so a qualified
			// column reference inside that ORDER BY must be rewritten here, not left
			// for buildEdgeSelect's own rewriteQualifiedRefs call to catch, or it
			// survives unrewritten inside _sl_ord for every hasKids=false, ordered
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
	// For a multi-hop count, joins/matchAlias let buildEdgeSelect JOIN
	// through the intermediate table(s) and quote SELECT/GROUP BY against the
	// FIRST intermediate table's alias instead of the FINAL table's — see
	// buildEdgeSelect's own colAlias comment. carried is discarded: a count
	// section never adds a SQL carried-key column (the SELECT stays "key
	// columns + count(*)"). For a single-hop count, edgeJoinChain returns
	// (nil, nil, srcAlias), so joins is a no-op and matchAlias equals srcAlias.
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
	// Right after root.cols()' NULL padding, NULL-pad the declared count
	// preload columns too (this branch is not a root row either).
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
// preserving materialization order without an SQL-level guarantee (an
// accepted limitation for a top-level root query without an ORDER BY).
// Two callers supply a non-empty parentOrdCol:
//
//   - runThen's route 4: sectionload builds the parent query itself via
//     buildEdgeSelect(collapseOrderBy: true), so when the edge requested an
//     ORDER BY, that query already carries an explicit, real ordinal column
//     of its own (ordCol).
//   - All/Build, for a caller-supplied top-level parentQuery that itself has
//     an ORDER BY: installParentOrdColumn (run.go) appends a row_number()
//     window ordered by that same ORDER BY onto the parent query, so
//     _sl_src's row_number() can follow it explicitly instead of the same
//     unguaranteed pass-through above.
//
// plan.parentOuterOrd/parentOuterAlias (run.go's outerParentOrd) is a third,
// preferred source for the same ordinality: when set, _sl_src's
// row_number() OVER (...) is placed OUTSIDE the parent subquery instead of
// relying on parentOrdCol's inside placement — see outerParentOrd's doc
// comment for why a parent LIMIT needs this to keep using a top-N sort.
func composeIsland(parent bob.Query, root partsSpec, plan islandPlan, parentOrdCol string) (bob.Query, error) {
	if err := checkDeclaredCountCollisions(root, plan); err != nil {
		return nil, err
	}
	if err := checkWidth(root, plan, DefaultMaxColumns); err != nil {
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
	// every "_sl_t"-aliased reference below.
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
		explicitCols := make([]any, 0, len(cols)+len(plan.declaredCounts))
		for _, c := range cols {
			explicitCols = append(explicitCols, psql.Quote("q", c.Name))
		}
		// The declared PreloadCount columns are on the parent's own
		// SELECT list too, and section0List reads them back by name off
		// _sl_src — the other two branches pass them through with .*, so
		// this explicit list must carry them as well, in the same order
		// section0List uses (root.cols() first, declaredCounts after).
		for _, dc := range plan.declaredCounts {
			explicitCols = append(explicitCols, psql.Quote("q", dc.Alias))
		}
		srcCols = explicitCols
		ordExpr = psql.Raw("row_number() OVER (ORDER BY " + quoteIdent("q") + "." + quoteIdent(parentOrdCol) + ") AS " + quoteIdent(colOrd))
	}
	selectCols := make([]any, 0, len(srcCols)+1)
	selectCols = append(selectCols, srcCols...)
	selectCols = append(selectCols, ordExpr)
	wrapped := psql.Select(
		sm.Columns(selectCols...),
		sm.From(parentSource{parent}).As(parentWrapAlias),
	)

	mods := []bob.Mod[*pdialect.SelectQuery]{
		sm.With(cteSrc).As(wrapped).Materialized(),
	}

	// CTEs for nodes that have children (they are referenced twice:
	// once to emit their section, once as the semi-join key source).
	// Route 1 of buildEdgeSelect. The edge's own options apply here — its CTE
	// body is the one physical SELECT for this edge — so both the section's
	// own rows (read back by unionBranch, route 3) and the semi-join
	// population for its own children/grandchildren see the same
	// filtered/limited/ordered set, matching bob's nested-ThenLoad LIMIT
	// semantics.
	//
	// hasKidsOrdered records, for every hasKids node, whether its opts turned
	// out to have an ORDER BY — derived from the SAME extractOptionsFragment
	// call below that builds this CTE body, and handed to unionBranch's route 3
	// later in this function instead of letting it re-extract the same opts a
	// second time (see unionBranch's hasKidsOrdered doc for what could go
	// wrong with two independent extractions of a stateful opts mod).
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
			// (spec.carried), not from `child` itself.
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
		// expr.Raw, not psql.Raw (the same failure class as the matchWhere/
		// parentKeySubquery case above): section0List joins column names and
		// NULL-cast DB type names verbatim, and a PostgreSQL quoted identifier may
		// legally contain "?" (e.g. Column{Name: "flag?"}). psql.Raw/expr.Clause's
		// "?" scanning would miscount it as a bob argument placeholder; expr.Raw
		// writes the string as-is with no scanning at all.
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

	// One more UNION ALL branch per count section (counts ride the SAME
	// island, never a separate query).
	for _, c := range plan.counts {
		branch, err := countBranch(root, plan, c)
		if err != nil {
			return nil, err
		}
		mods = append(mods, sm.UnionAll(branch))
	}

	return psql.Select(mods...), nil
}

// checkDeclaredCountCollisions checks that no declared alias collides
// with a root column, a section column, or an _sl_cnt<k> column, and that
// declared aliases do not duplicate each other. It needs a baseline set
// that excludes plan.declaredCounts itself, so it calls expectedColumns on
// a copy with declaredCounts emptied (islandPlan is a value type, so the
// caller's plan is not modified). [2:] drops the leading _sl_set/_sl_ord
// (these are quoteIdent'd reserved names that no user declaration can
// collide with, so they are not compared).
func checkDeclaredCountCollisions(root partsSpec, plan islandPlan) error {
	base := plan
	base.declaredCounts = nil
	existing := make(map[string]bool, len(base.sections)+len(base.counts))
	for _, name := range expectedColumns(root, base)[2:] {
		existing[name] = true
	}
	// Also check the declared list itself for duplicate aliases. Attaching the
	// same preloader twice puts two same-named columns in _sl_src's output,
	// and "_sl_src"."__count_X" becomes an ambiguous column reference error in
	// PostgreSQL (inferred, not measured: inside _sl_src, the outer-ordering
	// branch and the no-ORDER-BY branches expose all of the parent query's
	// columns via "<parent table>".* / q.*, and the inner-ordering branch
	// lists the same alias twice explicitly, so either path yields two columns
	// of the same name and the reference is ambiguous).
	// checkDeclaredCountCollisions rejects this path with the same error type.
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
	// Declared count preloads are not part of root.cols(); they are extra
	// columns specific to the parent branch. They already exist inside _sl_src
	// (the correlated-subquery columns PreloadCount added), so they only need
	// to be read back here by name.
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
