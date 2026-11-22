package sectionload

import "fmt"

// validatePlan walks the declared tree once — after the parent query
// hooks and before sectionload builds or executes its own query — and
// rejects two generator/wiring mistakes that would otherwise surface as
// a scan-time panic or a SQL "column does not exist" error deep inside
// execIsland:
//
//  1. An edge whose ColumnPlan restricts columns must set the *KeyFromBuf
//     override that node.go's parentKeyFromBufAny/childKeyFromBufAny
//     otherwise falls back to BuildFor(nil) for — a call that panics on
//     any NOT NULL column this ColumnPlan excluded (see node.go's
//     Edge.parentKeyFromBufAny doc for why).
//  2. A nested edge's join columns (FromCols) must be included in its
//     parent edge's own restricted column set, or the parent's generated
//     SQL never selects a column the nested edge's join condition
//     references.
//
// Both checks share the same shape — "if this edge's ColumnPlan
// restricts, check something about it and its children" — so one walk
// covers both. The walk visits every Node regardless of its then flag:
// a Then(...) boundary still carries its own children (Then only sets a
// flag, per node.go's Then function), and at runtime a Then boundary's
// child becomes a nested island's root via edgeRootParts, which is
// exactly the same "restricted buffer, same *FromBuf requirement" shape
// as an in-island section (execIsland's rootChildKeyFn is literally
// e.childKeyFromBufAny — see exec.go). So no separate recursion into
// runThen's nested islands is needed: the declared tree already contains
// every edge that will ever run, then-boundary or not.
func validatePlan(nodes []Node, declared []declaredCountPreload) error {
	declaredNames := make(map[string]bool, len(declared))
	for _, dc := range declared {
		declaredNames[dc.RelName] = true
	}

	// Only top-level count nodes are checked against the declared
	// PreloadCount names: PreloadCount attaches only to the root model,
	// so only a count node directly under the root can collide. A
	// same-named count node on a different table nested deeper in the
	// tree must not be flagged, so the check does not walk every depth.
	for _, n := range nodes {
		if n.count != nil && declaredNames[n.count.relName()] {
			return &PreloadCountConflictError{RelName: n.count.relName()}
		}
	}

	var walk func(n Node) error
	walk = func(n Node) error {
		if n.count != nil {
			// A count node carries no row-loading edgeSpec (n.edge is nil)
			// and is always a leaf (NewCountNode takes no children), so
			// neither of validatePlan's two checks — which are both about
			// an edgeSpec's ColumnPlan and its children — apply to it.
			// A multi-hop count edge DOES need the same structural
			// hop-chain invariants a multi-hop row edge needs —
			// validateHopChain covers exactly that subset. It ALSO needs
			// the count-edge counterpart of the Carried checks
			// (KeyCols/NewCarriedBuf, checked by validateMultiHopCountEdge
			// below): without it, a generator or fixture error leaving
			// NewCarriedBuf nil or KeyCols mis-sized would pass through
			// validatePlan undetected and panic later inside exec.go
			// (newCarriedBufAny()/carriedScanCols() called on a nil
			// interface or with a length assumption that no longer holds),
			// breaking the "All() rejects a broken multi-hop edge before
			// any query runs" guarantee validateMultiHopEdge already gives
			// row edges.
			if err := validateHopChain(n.count.sides(), n.count.relName()); err != nil {
				return err
			}
			return validateMultiHopCountEdge(n.count)
		}
		e := n.edge

		// Multi-hop structural checks. Independent of ColumnPlan —
		// a multi-hop edge with a ZERO ColumnPlan is the common case, and
		// still needs Carried/NewCarriedBuf/Sides validated before
		// composeIsland/execIsland reach them (validateMultiHopEdge's own
		// doc explains why this cannot live inside the
		// columnPlanRestricts() branch below).
		if err := validateMultiHopEdge(e); err != nil {
			return err
		}

		if e.columnPlanRestricts() {
			if err := validateRestrictedEdge(n); err != nil {
				return err
			}
		}
		for _, c := range n.children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range nodes {
		if err := walk(n); err != nil {
			return err
		}
	}
	return nil
}

// validateRestrictedEdge checks a row edge whose ColumnPlan restricts the
// selected columns: the *KeyFromBuf overrides that read the restricted
// buffer must exist, and every column those key readers (and the edge's
// children) need must still be fetched.
func validateRestrictedEdge(n Node) error {
	e := n.edge
	if len(e.sides()) == 1 {
		if !e.hasChildKeyFromBuf() {
			return &MissingKeyFromBufError{RelName: e.relName(), Field: "ChildKeyFromBuf", RestrictedRelName: e.relName()}
		}
	} else if !e.hasParentKeyFromBuf() {
		// A multi-hop edge's own child key never reaches this
		// edge's ChildKeyFromBuf at all — childKeyFrom (node.go)
		// reads the SEPARATE carried buffer via ParentKeyFromBuf
		// instead, even though ParentKeyFromBuf is nominally "the
		// parent's own key extractor" (it operates on a Buf[P], and
		// the carried buffer IS a Buf[P] — see Edge.Carried's doc).
		// Carried/NewCarriedBuf's presence is already guaranteed for
		// every multi-hop edge, restricted or not, by
		// validateMultiHopEdge above; requiring ChildKeyFromBuf here
		// would reject every restricted multi-hop edge
		// unconditionally, since generated code never sets it for a
		// multi-hop Edge (01_edges.go.tpl only emits
		// ParentKeyFromBuf when len(Sides) > 1).
		return &MissingKeyFromBufError{RelName: e.relName(), Field: "ParentKeyFromBuf", RestrictedRelName: e.relName()}
	}
	fetched := make(map[string]bool, len(e.childCols()))
	for _, c := range e.childCols() {
		fetched[c.Name] = true
	}
	// Check that this edge's own ToCols are
	// fetched. If this edge's ChildKeyFromBuf reads one of its toCols,
	// and we don't fetch it, the buffer-read fails silently with a
	// zero-value key; the stitching becomes a no-op. Detect it here,
	// before the sectioned query is built or executed.
	//
	// Scoped to a single-hop edge (len(sides())==1) only. For a
	// multi-hop edge, toCols()==sides()[0].ToCols names a column of
	// the FIRST INTERMEDIATE table, not the final child table
	// childCols() enumerates — it is never fetched by this edge's own
	// SQL (it is read via the carried buffer instead, from an
	// entirely different table), so this check does not apply and
	// would always spuriously fail for a correctly-built multi-hop
	// edge.
	if len(e.sides()) == 1 {
		for _, col := range e.toCols() {
			if !fetched[col] {
				return &MissingRequiredColumnError{
					RelName:      e.relName(),
					ChildRelName: e.relName(),
					Column:       col,
				}
			}
		}
	}
	for _, c := range n.children {
		// A count child has no edgeSpec (c.edge is nil), but its key
		// extraction against THIS edge's restricted parent buffer is
		// CountEdge.parentKeyFromBufAny, whose BuildFor(nil) fallback
		// has the same panic-on-excluded-columns failure mode as a row
		// child's — so it gets the same two checks, via countEdgeSpec.
		if c.count != nil {
			if !c.count.hasParentKeyFromBuf() {
				return &MissingKeyFromBufError{RelName: c.count.relName(), Field: "ParentKeyFromBuf", RestrictedRelName: e.relName()}
			}
			for _, col := range c.count.fromCols() {
				if !fetched[col] {
					return &MissingRequiredColumnError{
						RelName:      e.relName(),
						ChildRelName: c.count.relName(),
						Column:       col,
					}
				}
			}
			continue
		}
		if !c.edge.hasParentKeyFromBuf() {
			return &MissingKeyFromBufError{RelName: c.edge.relName(), Field: "ParentKeyFromBuf", RestrictedRelName: e.relName()}
		}
		for _, col := range c.edge.fromCols() {
			if !fetched[col] {
				return &MissingRequiredColumnError{
					RelName:      e.relName(),
					ChildRelName: c.edge.relName(),
					Column:       col,
				}
			}
		}
	}
	return nil
}

// validateHopChain checks the structural invariants of a hop chain that
// BOTH a row edge and a count edge must hold, independent of whether
// the edge carries a value alongside each row (Carried, row edges only —
// see validateMultiHopEdge for that half). These are items 3, 4 and 5
// of validateMultiHopEdge's doc comment:
//
//   - every hop's FromCols/ToCols are non-empty and of equal length
//     (item 3);
//   - the hop-to-hop table chain is unbroken (item 4);
//   - the hop-to-hop schema chain is unbroken (item 5).
func validateHopChain(sides []Side, relName string) error {
	if len(sides) <= 1 {
		return nil
	}
	for i, s := range sides {
		if len(s.FromCols) == 0 || len(s.ToCols) == 0 {
			return &CarriedKeyMismatchError{
				RelName: relName,
				Reason: fmt.Sprintf(
					"Sides[%d] has empty FromCols/ToCols (len=%d/%d); every hop needs at least one join column",
					i, len(s.FromCols), len(s.ToCols),
				),
			}
		}
		if len(s.FromCols) != len(s.ToCols) {
			return &CarriedKeyMismatchError{
				RelName: relName,
				Reason: fmt.Sprintf(
					"Sides[%d].FromCols has %d column(s) but Sides[%d].ToCols has %d; edgeJoinChain pairs them positionally and requires equal length",
					i, len(s.FromCols), i, len(s.ToCols),
				),
			}
		}
	}
	for i := 1; i < len(sides); i++ {
		if sides[i-1].ToTable != sides[i].FromTable {
			return &CarriedKeyMismatchError{
				RelName: relName,
				Reason: fmt.Sprintf(
					"Sides[%d].ToTable (%q) != Sides[%d].FromTable (%q); the hop chain is broken",
					i-1, sides[i-1].ToTable, i, sides[i].FromTable,
				),
			}
		}
		if sides[i-1].ToSchema != sides[i].FromSchema {
			return &CarriedKeyMismatchError{
				RelName: relName,
				Reason: fmt.Sprintf(
					"Sides[%d].To (%s) != Sides[%d].From (%s); the hop chain's schema is broken",
					i-1, schemaTableLabel(sides[i-1].ToSchema, sides[i-1].ToTable),
					i, schemaTableLabel(sides[i].FromSchema, sides[i].FromTable),
				),
			}
		}
	}
	return nil
}

// validateMultiHopEdge checks the generation-contract invariants a
// multi-hop Edge (len(sides()) > 1) must hold, before composeIsland/
// execIsland reach them deep inside SQL building or row scanning. It
// checks the Carried-specific invariants itself (items 1 and 2) and
// delegates the hop-chain invariants (items 3, 4 and 5) to
// validateHopChain, which count edges share (validatePlan's n.count
// branch calls validateHopChain directly, since a CountEdge never has
// Carried/NewCarriedBuf-as-edgeSpec). When an edge violates several
// invariants at once, the hop-chain error is reported before the
// Carried-specific one.
//
//  1. len(Carried) == len(Sides[0].FromCols): edgeJoinChain (compose.go)
//     and carriedScanCols() (node.go) both index Carried and
//     Sides[0].{From,To}Cols positionally, assuming equal length; a
//     mismatch would silently truncate the carried-key SELECT list or
//     panic with an out-of-range index deep inside either.
//  2. NewCarriedBuf != nil: without it, execIsland has no Buf[P] to scan
//     the carried column(s) into at all, and childKeyFrom's multi-hop
//     branch would panic on the nil interface type assertion.
//  3. len(Sides[i].FromCols) == len(Sides[i].ToCols) > 0 for every side:
//     edgeJoinChain (compose.go) walks s.ToCols and indexes
//     s.FromCols[j] positionally, with no bounds check, and
//     carriedScanCols() (node.go) does the same against
//     Sides[0].FromCols[i]. A length mismatch (or a hop with zero
//     columns) would panic with an out-of-range index deep inside SQL
//     building/scanning, or silently build a malformed JOIN condition,
//     instead of failing loudly here before any query is built.
//  4. Sides[i-1].ToTable == Sides[i].FromTable for every i>=1: bob's own
//     Relationship.Validate() (orm/relationship.go) guarantees this for
//     every Relationship it produces, so a violation here means a
//     hand-written or hand-patched Edge (a test fixture, or a manually
//     configured relation) broke the chain — edgeJoinChain's alias
//     scheme (compose.go) is built entirely on this invariant holding.
//  5. Sides[i-1].ToSchema == Sides[i].FromSchema for every i>=1:
//     checking only the table name half of a hop's identity would accept
//     a schema mismatch — two same-named tables in different schemas are
//     different tables, so without this check a hand-written or
//     hand-patched Edge could silently JOIN into the wrong physical
//     table across a schema boundary. FromSchema/ToSchema may
//     legitimately both be "" (bob's psql driver omits "public" — Side's
//     own doc comment), so this compares them exactly the same way, with
//     no special-casing of the empty string.
//
// This check is intentionally run for every multi-hop edge, outside the
// columnPlanRestricts() branch in validatePlan: a multi-hop edge with a
// ZERO ColumnPlan (the common case — most through relationships select
// every column) still needs every invariant held, and
// columnPlanRestricts() is false for it. Nesting this check inside that
// branch would silently skip it for exactly the edges most likely to
// exist.
func validateMultiHopEdge(e edgeSpec) error {
	if err := validateHopChain(e.sides(), e.relName()); err != nil {
		return err
	}
	sides := e.sides()
	if len(sides) <= 1 {
		return nil
	}
	if got, want := len(e.carriedCols()), len(sides[0].FromCols); got != want {
		return &CarriedKeyMismatchError{
			RelName:  e.relName(),
			Carried:  got,
			FromCols: want,
			Reason:   "len(Carried) must equal len(Sides[0].FromCols)",
		}
	}
	if e.newCarriedBufAny() == nil {
		return &CarriedKeyMismatchError{
			RelName: e.relName(),
			Reason:  "NewCarriedBuf must be set for a multi-hop edge (len(Sides) > 1)",
		}
	}
	return nil
}

// validateMultiHopCountEdge is the count-edge counterpart of
// validateMultiHopEdge's Carried checks (items 1 and 2) above: a
// multi-hop count edge's KeyCols plays exactly the role Carried plays
// for a row edge (both are sized against Sides[0].FromCols and both
// exist so exec.go can scan the grouped key back into the SAME Go type
// the parent's own join key uses), and NewCarriedBuf is the identical
// factory field by the identical name. Without this check, a generator
// or fixture error leaving CountEdge.KeyCols empty/mis-sized or
// NewCarriedBuf nil would pass through validatePlan undetected
// (validateHopChain alone only checks the Sides chain itself, never
// KeyCols/NewCarriedBuf) and panic later inside exec.go
// (newCarriedBufAny() called on a nil interface, or carriedScanCols()
// indexing past KeyCols/Sides[0].FromCols's shorter length).
//
// The error type reused here is CarriedKeyMismatchError — NOT
// CountKeyColsMismatchError (errors.go, used by countKeyCols/node.go to
// compare KeyCols against ToCols at query-build time) — because this
// check's shape is symmetric with validateMultiHopEdge's Carried checks
// (same struct, same two violation kinds, same
// RelName/Carried/FromCols/Reason fields, checked at the same
// validatePlan call site, against the same Sides[0].FromCols length),
// not with countKeyCols's KeyCols-vs-ToCols shape.
func validateMultiHopCountEdge(e countEdgeSpec) error {
	sides := e.sides()
	if len(sides) <= 1 {
		return nil
	}
	if got, want := len(e.declaredKeyCols()), len(sides[0].FromCols); got != want {
		return &CarriedKeyMismatchError{
			RelName:  e.relName(),
			Carried:  got,
			FromCols: want,
			Reason:   "len(KeyCols) must equal len(Sides[0].FromCols)",
		}
	}
	if e.newCarriedBufAny() == nil {
		return &CarriedKeyMismatchError{
			RelName: e.relName(),
			Reason:  "NewCarriedBuf must be set for a multi-hop count (len(Sides) > 1)",
		}
	}
	return nil
}

// schemaTableLabel formats a Side's schema/table pair for an
// error message as "schema.table", or just "table" when schema is empty
// (bob's psql driver omits "public" — the same convention Side's own doc
// comment describes), so a schema-mismatch error names both tables
// unambiguously without printing a spurious leading dot for the common
// (no explicit schema) case.
func schemaTableLabel(schema, table string) string {
	if schema == "" {
		return table
	}
	return schema + "." + table
}
