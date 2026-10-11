package sectionload

import (
	"fmt"
	"strings"
)

// ReservedNameError: the parent query uses one of the reserved
// identifiers (_sl_set / _sl_ord / _sl_src / _sl_n<k> / _sl_cnt<k>).
type ReservedNameError struct {
	Name string
}

func (e *ReservedNameError) Error() string {
	return fmt.Sprintf("sectionload: parent query uses reserved identifier %q; rename it", e.Name)
}

// TooWideError: the combined query would exceed the column guardrail.
// Split the tree with sectionload.Then(...) at or above AtRel.
type TooWideError struct {
	Total int
	Max   int
	AtRel string
}

func (e *TooWideError) Error() string {
	return fmt.Sprintf(
		"sectionload: %d columns exceed the limit of %d (crossed while adding %s); split the tree with sectionload.Then(...)",
		e.Total, e.Max, e.AtRel,
	)
}

// ColumnSelectionConflictError: NewColumnPlan was called with both only
// and except non-empty. They are mutually exclusive selection modes.
type ColumnSelectionConflictError struct{}

func (e *ColumnSelectionConflictError) Error() string {
	return "sectionload: only and except are mutually exclusive"
}

// UnknownKeyColumnError: a key column name (NewColumnPlan's keyCols, or a
// CountEdge's ToCols resolved by countKeyCols) does not name any column in
// the table's full column list. This indicates a caller/generator error (a
// stale or misspelled key column name), not a user-facing column-selection
// mistake.
type UnknownKeyColumnError struct {
	Column string
}

func (e *UnknownKeyColumnError) Error() string {
	return fmt.Sprintf("sectionload: key column %q not found on table", e.Column)
}

// CountKeyColsMismatchError: a multi-hop CountEdge's generator-
// supplied KeyCols does not match its own ToCols — either in length, or
// in per-position name/order:
// KeyCols[i].Name must equal Sides[0].ToCols[i] for every i, because
// countKeyCols (node.go) trusts KeyCols' Name verbatim as the SQL
// SELECT/GROUP BY column against the FIRST intermediate table — a
// same-length-but-wrong-name-or-order KeyCols would build a query that
// still runs successfully (the wrong name may well be a real column of
// that table) but groups/matches on the WRONG column, silently
// misattributing counts to the wrong parent or zeroing every parent out.
// Reason carries which of the two violations fired; Declared/ToCols stay
// populated for both (length-mismatch's own message; a name/order
// mismatch's message is carried entirely in Reason instead).
type CountKeyColsMismatchError struct {
	RelName  string
	Declared int
	ToCols   int
	Reason   string
}

func (e *CountKeyColsMismatchError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("sectionload: %s: %s", e.RelName, e.Reason)
	}
	return fmt.Sprintf(
		"sectionload: %s: CountEdge.KeyCols has %d column(s) but ToCols has %d; the generator must emit exactly one KeyCols entry per Sides[0].ToCols entry",
		e.RelName, e.Declared, e.ToCols,
	)
}

// KeyColumnExcludedError: the caller named a join key column in an
// Except(...) column selection. sectionload always fetches join key
// columns for stitching regardless of the caller's selection;
// naming one in Except is rejected rather than silently
// ignored, kept as Internal, or allowed to break stitching.
type KeyColumnExcludedError struct {
	Column string
}

func (e *KeyColumnExcludedError) Error() string {
	return fmt.Sprintf(
		"sectionload: %q is a join key column and cannot be excluded; it is always fetched for stitching (omit it from Except, or use Only to select the columns you want)",
		e.Column,
	)
}

// MissingKeyFromBufError: an edge's ColumnPlan restricts which columns
// are fetched, but the *KeyFromBuf override this restriction requires was
// not set by the generator. Field names the missing struct field
// (ParentKeyFromBuf or ChildKeyFromBuf) on the named relation's Edge.
// RestrictedRelName names the relation whose scan buffer is restricted —
// the one the missing override would otherwise be read against. It is
// the SAME relation as RelName when the edge's own ChildKeyFromBuf is
// missing (the edge restricts its own buffer), but a DIFFERENT relation
// (the parent edge) when a child's ParentKeyFromBuf is missing (the
// child reads its PARENT's restricted buffer, not its own).
// Without it, the runtime falls back to rebuilding the full model from a
// buffer that was only partially scanned; the generated Buf.BuildFor(nil)
// unconditionally calls .MustGet() on every NOT NULL column of the
// table, which panics for any column this ColumnPlan excluded. Checked
// once, statically, after the parent query hooks and before sectionload
// builds or executes its own query (validatePlan in validate.go), so
// this surfaces as a returned error instead of a scan-time panic in
// execIsland.
type MissingKeyFromBufError struct {
	RelName           string
	Field             string
	RestrictedRelName string
}

func (e *MissingKeyFromBufError) Error() string {
	return fmt.Sprintf(
		"sectionload: relation %q has nil Edge.%s, but it is evaluated against "+
			"relation %q's restricted scan buffer; the generator must set %s",
		e.RelName, e.Field, e.RestrictedRelName, e.Field,
	)
}

// MissingRequiredColumnError: a required join column is missing from a
// restricted edge's own ColumnPlan. validate.go constructs it from two
// different call sites, distinguished by whether RelName == ChildRelName:
//
//  1. RelName == ChildRelName (the edge names itself on both sides): the
//     edge's OWN ToCols are read by its OWN ChildKeyFromBuf against its
//     OWN restricted scan buffer (validatePlan's check of e.toCols()
//     against e.childCols()). If a ToCol here is not fetched, the buffer
//     read does not error — it silently returns a Go zero value, so
//     stitching becomes a no-op instead of failing loudly.
//  2. RelName != ChildRelName: a nested edge's join column (FromCols) is
//     not included in its PARENT edge's own restricted column set
//     (ColumnPlan) — RelName is the parent, ChildRelName the nested
//     edge. The parent edge's ColumnPlan can be constructed without
//     going through NewColumnPlan (which would have added the column
//     automatically as Internal), so nothing else prevents this. Left
//     unchecked, the generated SQL for the nested edge references a
//     column the parent's CTE/subquery never selected — a SQL-time
//     "column does not exist" error.
//
// Column names the missing column in both cases. Checked once,
// statically, after the parent query hooks and before sectionload builds
// or executes its own query (validatePlan in validate.go).
type MissingRequiredColumnError struct {
	RelName      string
	ChildRelName string
	Column       string
}

func (e *MissingRequiredColumnError) Error() string {
	if e.RelName == e.ChildRelName {
		// Case 1 above: RelName and ChildRelName are the same relation,
		// so naming it twice ("relation X needs column Y on X") would be
		// confusing/redundant. Mention it once and speak of "its own"
		// column and ColumnPlan instead.
		return fmt.Sprintf(
			"sectionload: relation %q needs its own column %q for its join, but its ColumnPlan "+
				"does not fetch it; build %q's ColumnPlan with NewColumnPlan and pass %q as one "+
				"of keyCols",
			e.RelName, e.Column, e.RelName, e.Column,
		)
	}
	// Case 2 above: RelName (parent) and ChildRelName (nested edge) differ,
	// so both names are needed and this wording is unchanged.
	return fmt.Sprintf(
		"sectionload: relation %q needs column %q on %q for its join, but %q's ColumnPlan "+
			"does not fetch it; build %q's ColumnPlan with NewColumnPlan and pass %q as one "+
			"of keyCols",
		e.ChildRelName, e.Column, e.RelName, e.RelName, e.RelName, e.Column,
	)
}

// CountNodeThenError: sectionload.Then(...) was applied to a count node.
// Then marks a query boundary — "load the subtree below this node in its
// own round trip" — and a count node has no subtree: it loads no models,
// only a number per parent. Accepting the marker would either silently do
// nothing (leaving the caller believing an extra round trip was requested)
// or hand the Then machinery an edge that cannot build rows, so it is
// rejected while that specific island's node list is being planned by
// planIsland — before its sectioned UNION ALL query is composed or
// executed. The same planIsland-recursion caveat applies here: a count
// node nested under a Then boundary that is itself nested inside another
// Then's children is only reached when runThen (hooks.go) calls
// planIsland on those children at runtime; Build never calls runThen and
// cannot detect such a violation at all.
type CountNodeThenError struct {
	RelName string
}

func (e *CountNodeThenError) Error() string {
	return fmt.Sprintf(
		"sectionload: Then(...) cannot be applied to the count node for relationship %q; a count node loads no rows, so it is always a leaf of its island",
		e.RelName,
	)
}

// ThenBoundaryFromWhereError: sectionload.Then(...) was applied to an edge
// whose relationship carries a fixed condition on the FROM (parent) side.
//
// Inside one island the condition is applied by narrowing the parent-key
// semi-join: the parent's rows live in a CTE belonging to the same query,
// so parentKeySubquery (compose.go) emits
// "IN (SELECT <fromCols> FROM <parent CTE> WHERE <from condition>)" and
// only parents satisfying it contribute a key. A Then(...) boundary has no
// such CTE — runThen collects the parent keys into Go values and matchWhere
// matches them as a literal row-value IN list, a branch that never reaches
// parentKeySubquery. The condition would therefore be dropped silently, and
// children would be stitched onto parents that do not satisfy it.
//
// Rejecting is deliberate. bob's own handling of a parent-most from-side
// condition is not consistent enough to copy (its single-hop count template
// omits the condition entirely, while its relationship-query template emits
// a reference to a table that is not in the query's FROM clause), so there
// is no "correct" behavior to reproduce here. Following the rule —
// refuse rather than quietly return a wrong result — the tree is rejected
// while that specific island's node list is being planned by planIsland.
//
// That guarantee is only as strong as planIsland's own recursion, though:
// for a Then boundary declared directly under All/Build's own nodes, this
// runs before anything else does. For one nested inside another Then's own
// children, planIsland only walks it once runThen (hooks.go) actually
// reaches that boundary at runtime — and is skipped entirely for a run
// whose matched parent keys are empty. Build never calls runThen, so it
// cannot see a violation nested this way at all; this is the same
// planIsland-recursion caveat CountNodeThenError above already carries.
//
// For a multi-hop edge, this rejection applies ONLY to Sides[0]'s
// own "from" condition (the parent-key subquery narrowing described
// above) — not to a from-side condition on an intermediate hop
// (Sides[i], i>=1). A side i>=1 condition is rendered inside that hop's
// own JOIN ON clause (edgeJoinChain, compose.go), and a Then boundary's
// route-4 parent query builds that JOIN chain the exact same way routes
// 1/2 do; it never goes anywhere near parentKeySubquery, so nothing about
// it is dropped by crossing a Then boundary. planIsland's own check
// (node.go) only ever sees Sides[0].Where in the first place (via
// edgeWhere(), which already means sides()[0].Where), so this scoping
// falls out of that derivation rather than needing a second, separate
// side-aware check here.
type ThenBoundaryFromWhereError struct {
	RelName string
}

func (e *ThenBoundaryFromWhereError) Error() string {
	return fmt.Sprintf(
		"sectionload: Then(...) cannot be applied to relationship %q because it has a fixed condition on the from (parent) side; a Then boundary matches parent keys collected as literal values, so there is no parent CTE to constrain the way the same-island semi-join does; keep this relation in the same island, or remove the from-side condition",
		e.RelName,
	)
}

// StaleCountContractError: a generated count node called the model's
// PreloadCount(name, count) and the corresponding C.<Rel> field was still
// nil afterwards.
//
// bob's generated PreloadCount is a switch with NO default case, so a name
// it does not know is a silent no-op: the query cost is paid and the
// caller still sees "not loaded". The generated count node therefore
// checks the field it just set — a typed field access, not reflection
// — and reports this instead. In practice it means the generated
// sectionload package and the generated models package were generated
// from different schema states and must be regenerated together.
type StaleCountContractError struct {
	RelName string
}

func (e *StaleCountContractError) Error() string {
	return fmt.Sprintf(
		"sectionload: PreloadCount(%q) left the count field unset; the generated sectionload package and the generated models package are out of sync — regenerate both",
		e.RelName,
	)
}

// LayoutError: the executed query returned columns that do not match the
// generated parts. Usually the parent query does not select the table's
// full column set (unsupported), or the generated package is stale.
type LayoutError struct {
	Expected  []string
	Got       []string
	FirstDiff int
}

func (e *LayoutError) Error() string {
	return fmt.Sprintf(
		"sectionload: result columns do not match the generated parts (first difference at %d):\n  expected: %s\n  got:      %s",
		e.FirstDiff, strings.Join(e.Expected, ", "), strings.Join(e.Got, ", "),
	)
}

// SelectOptionsStructureError: a caller-provided SelectOptions mod changed
// something outside what sectionload can absorb into a fixed section
// layout — the SELECT column list, FROM/JOIN, a CTE, or a combine
// (UNION/INTERSECT/EXCEPT) — for relationship RelName. Detected before the
// SQL is ever sent to the database: a mod is
// free to add WHERE/ORDER BY/LIMIT/OFFSET/GROUP BY/HAVING conditions, but
// not to change what columns come back or where they come from, since
// every other section's row in the same UNION ALL depends on this
// section's column list staying exactly what the generated parts declare.
type SelectOptionsStructureError struct {
	RelName string
	Field   string // "select columns" | "FROM/JOIN" | "WITH (CTE)" | "combine (UNION/INTERSECT/EXCEPT)"
}

func (e *SelectOptionsStructureError) Error() string {
	return fmt.Sprintf(
		"sectionload: SelectOptions for relationship %q changed %s, which sectionload cannot absorb into a fixed section layout; only WHERE/ORDER BY/LIMIT/OFFSET-style mods are supported",
		e.RelName, e.Field,
	)
}

// UnsupportedMapperModsError: the parent query carries one or more
// bob.MapperMod values (e.g. a JOIN Preload, or a PreloadCount) that
// sectionload does not consume. Left unchecked, these currently succeed
// silently: the extra columns/JOIN they add land inside sectionload's own
// _sl_src wrapping, but the generated parts' fixed column list never
// selects them back out, so the caller gets no data and no error at all
// (a measured behavior, not the LayoutError one might guess at).
// guessed at). Returning this error instead moves the relation onto its
// own sectionload node.
type UnsupportedMapperModsError struct {
	Count int
}

func (e *UnsupportedMapperModsError) Error() string {
	return fmt.Sprintf(
		"sectionload: parent query carries %d mapper mod(s) sectionload cannot apply (e.g. a JOIN Preload or PreloadCount); move the relation to a sectionload node instead",
		e.Count,
	)
}

// UncheckableParentQueryError reports a parent query whose outer dynamic
// type does not expose its PostgreSQL SelectQuery. Such a wrapper may
// still forward WriteQuery to an inner query carrying contextual mods, so
// silently skipping the build-time MapperMod check would be unsafe.
type UncheckableParentQueryError struct {
	Type string
}

func (e *UncheckableParentQueryError) Error() string {
	return fmt.Sprintf(
		"sectionload: parent query type %s does not expose its PostgreSQL select query; pass the concrete psql query directly",
		e.Type,
	)
}

// CarriedKeyMismatchError: a multi-hop Edge's generation contract
// was violated in a way validatePlan (validate.go) checks
// statically for a declared tree — Carried/NewCarriedBuf missing or the
// wrong length, or Sides not actually chained — or (see
// stitch.go's stitchInto) a multi-hop edge reached the
// "no precomputed child key" fallback path, which for a single-hop edge
// safely falls back to the child model's own ChildKey but for a
// multi-hop edge would silently stitch using the FINAL table's columns —
// not the parent's actual join key (the final
// table has no column holding it at all). Carried/FromCols are filled in
// by validatePlan's length-mismatch checks; Reason is a short,
// human-readable label distinguishing which contract was violated (also
// used stand-alone, with Carried/FromCols left at zero, by the
// stitch.go check that has no lengths to report).
type CarriedKeyMismatchError struct {
	RelName  string
	Carried  int
	FromCols int
	Reason   string
}

func (e *CarriedKeyMismatchError) Error() string {
	// Carried/FromCols are populated (validate.go's length-mismatch
	// check), so they are rendered too: the concrete counts a caller
	// needs to diagnose "how many did I have vs. want" would otherwise be
	// invisible in the returned error text — only Reason's
	// human-readable label would be. Both callers that leave
	// Carried/FromCols at zero (NewCarriedBuf missing, chain broken) are
	// unaffected: their Reason string already states the violated
	// contract on its own, and printing "carried=0, fromCols=0" alongside
	// it does not contradict anything they say.
	return fmt.Sprintf(
		"sectionload: relation %q: multi-hop carried-key contract violated (%s) [carried=%d, fromCols=%d]",
		e.RelName, e.Reason, e.Carried, e.FromCols,
	)
}

// ColumnSelectionEmptyError: SelectOptions.Only or Except was called with
// zero column names. Distinct from "never called" (both stay unset and
// mean no restriction): an explicit call with an empty list is treated as
// a caller mistake, not a no-op, so it is rejected rather than silently
// behaving like Only/Except was never used.
type ColumnSelectionEmptyError struct {
	RelName string
	Field   string // "Only" | "Except"
}

func (e *ColumnSelectionEmptyError) Error() string {
	return fmt.Sprintf(
		"sectionload: relationship %q called SelectOptions.%s() with no column names; omit the call entirely for no restriction, or name at least one column",
		e.RelName, e.Field,
	)
}

// UnknownColumnError: SelectOptions.Only/Except named a column that does
// not exist on the table this edge fetches from. Distinct from
// UnknownKeyColumnError (parts.go), which reports a generator/keyCols
// contract error: this reports a caller-facing typo in a column list the
// caller supplied directly.
type UnknownColumnError struct {
	RelName string
	Field   string // "Only" | "Except"
	Column  string
}

func (e *UnknownColumnError) Error() string {
	return fmt.Sprintf(
		"sectionload: relationship %q: SelectOptions.%s named column %q, which is not a column of this relationship's table",
		e.RelName, e.Field, e.Column,
	)
}

// CountColumnRestrictionUnsupportedError: SelectOptions.Only/Except was
// passed to a count node (RelCount). A count section's SELECT is always
// exactly the join key columns plus count(*) (CountEdge has no
// ColumnPlan field at all — see parts.go) — there is no model column to
// restrict, so silently accepting Only/Except would make the caller
// believe a restriction took effect when nothing changed (sectionload
// never silently no-ops a caller's explicit instruction).
type CountColumnRestrictionUnsupportedError struct {
	RelName string
}

func (e *CountColumnRestrictionUnsupportedError) Error() string {
	return fmt.Sprintf(
		"sectionload: relationship %q is a count node; SelectOptions.Only/Except has no effect on a count's fixed key+count(*) SELECT and is rejected rather than silently ignored",
		e.RelName,
	)
}

// DuplicateColumnError: SelectOptions.Only or Except named the same
// column more than once in a single call (e.g. Except("a", "a")). A
// caller-facing input mistake, checked at resolution time alongside
// ColumnSelectionConflictError/ColumnSelectionEmptyError/
// UnknownColumnError (naming the same column twice on one side must be
// an explicit error).
type DuplicateColumnError struct {
	RelName string
	Field   string // "Only" | "Except"
	Column  string
}

func (e *DuplicateColumnError) Error() string {
	return fmt.Sprintf(
		"sectionload: relationship %q: SelectOptions.%s named column %q more than once",
		e.RelName, e.Field, e.Column,
	)
}

// PreloadCountConflictError: both a CountEdge (NewCountNode) and a bob
// PreloadCount declaration were specified for the same relationship. The
// two are independent paths writing to the same C.<Rel> field, so which
// one leaves the final value would be undefined; this is rejected with
// an explicit error before execution. The comparison is limited to
// top-level count nodes: PreloadCount attaches only to the root model,
// so only a count node directly under the root can collide, and a
// same-named count node on a different nested table is not considered.
type PreloadCountConflictError struct {
	RelName string
}

func (e *PreloadCountConflictError) Error() string {
	return fmt.Sprintf(
		"sectionload: relationship %q has both a CountEdge (NewCountNode) and a bob PreloadCount declared on the parent query; both write to the same C.%s field, so this is rejected rather than leaving the outcome undefined",
		e.RelName, e.RelName,
	)
}

// DeclaredCountColumnConflictError: the column alias that bob's
// PreloadCount attached collides with a column name fixed by the
// generated contract (root columns, section columns, _sl_cnt<k>) or
// with another PreloadCount declaration's own alias.
type DeclaredCountColumnConflictError struct{ Alias string }

func (e *DeclaredCountColumnConflictError) Error() string {
	return fmt.Sprintf("sectionload: declared count preload column %q collides with a fixed layout column or another declared count preload", e.Alias)
}

// MultiHopColumnRestrictionUnsupportedError: SelectOptions.Only/Except
// was passed to a multi-hop (through) edge (len(Edge.Sides) > 1). A
// multi-hop edge's own child-side join key (Edge.toCols(), which is
// sides()[0].ToCols) names a column of the FIRST INTERMEDIATE table,
// never the final child table a ColumnPlan would restrict — the final
// table's own ToColumns are never read for stitching at all (childKeyFrom
// reads the carried buffer instead — node.go's childKeyFrom doc). There is
// therefore no column on the final table NewColumnPlan can be told to
// protect for stitching, and computing a ColumnPlan with an empty keyCols
// would silently open a path where the caller could exclude a column
// stitching secretly depends on — rejected outright instead.
// Column restriction on multi-hop edges is not supported today.
type MultiHopColumnRestrictionUnsupportedError struct {
	RelName string
}

func (e *MultiHopColumnRestrictionUnsupportedError) Error() string {
	return fmt.Sprintf(
		"sectionload: relationship %q is a multi-hop (through) edge; SelectOptions.Only/Except is not supported for multi-hop edges (no correct join-key protection can be computed for the final table) and is rejected rather than silently risking broken stitching",
		e.RelName,
	)
}
