package sectionload

// A relationship's fixed condition on the FROM (parent) side cannot
// cross a Then(...) boundary.
//
// Inside one island, matchWhere emits the parent match as a semi-join
// against the parent's CTE, and parentKeySubquery (compose.go) narrows
// that subquery with the from-side condition:
//
//	... IN (SELECT <fromCols> FROM <parent CTE> WHERE <from condition>)
//
// A Then(...) boundary has no parent CTE. runThen collects the parent keys
// into Go values and matchWhere matches them as a literal row-value IN
// list, a branch that never reaches parentKeySubquery — so the from-side
// condition is silently dropped and children get stitched onto parents
// that do not satisfy it.
//
// planIsland therefore rejects the tree instead of building a query whose
// result would be quietly wrong. These tests pin the three cases that
// decide whether the check is correctly scoped: it must fire for a
// from-side condition at a Then boundary, and must NOT fire for the same
// condition inside one island, nor for a to-side condition at a Then
// boundary (to-side conditions are applied on every route).

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
)

// --- fixtures ------------------------------------------------------

type thenFromWhereParent struct {
	ID int
}

type thenFromWhereChild struct {
	ID       int
	ParentID int
}

type thenFromWhereChildBuf struct {
	id, parentID int
}

func (b *thenFromWhereChildBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.parentID} }

func (b *thenFromWhereChildBuf) BuildFor(modelCols []Column) *thenFromWhereChild {
	return &thenFromWhereChild{ID: b.id, ParentID: b.parentID}
}

func thenFromWhereChildParts() TableParts[*thenFromWhereChild] {
	return TableParts[*thenFromWhereChild]{
		Table: "tfw_children",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "parent_id", DBType: "int4"},
		},
		NewBuf: func() Buf[*thenFromWhereChild] { return &thenFromWhereChildBuf{} },
	}
}

// thenFromWhereEdge builds the edge under test. where is spliced in verbatim so a
// caller can vary only the side ("from" vs "to") without duplicating the
// fixture.
func thenFromWhereEdge(where []EdgeWhere) Edge[*thenFromWhereParent, *thenFromWhereChild] {
	return Edge[*thenFromWhereParent, *thenFromWhereChild]{
		RelName:   "Children",
		FromCols:  []string{"id"},
		ToCols:    []string{"parent_id"},
		ToMany:    true,
		Where:     where,
		Child:     thenFromWhereChildParts(),
		ParentKey: func(p *thenFromWhereParent) []any { return []any{p.ID} },
		ChildKey:  func(c *thenFromWhereChild) []any { return []any{c.ParentID} },
		Stitch:    func(p *thenFromWhereParent, cs []*thenFromWhereChild) error { return nil },
	}
}

// --- tests ---------------------------------------------------------

// A from-side condition at a Then(...) boundary must be rejected at plan
// time, before any SQL is composed.
func TestPlanIsland_ThenBoundaryWithFromWhereIsRejected(t *testing.T) {
	edge := thenFromWhereEdge([]EdgeWhere{
		{On: "from", Column: "is_active", SQLValue: "true"},
	})

	_, err := planIsland([]Node{Then(NewNode(edge))})
	if err == nil {
		t.Fatalf("planIsland must reject a Then boundary with a from-side condition, but returned no error")
	}

	var want *ThenBoundaryFromWhereError
	if !errors.As(err, &want) {
		t.Fatalf("err = %T (%v), want *ThenBoundaryFromWhereError", err, err)
	}
	if want.RelName != "Children" {
		t.Fatalf("RelName = %q, want %q", want.RelName, "Children")
	}
}

// The SAME from-side condition inside one island (no Then) must still be
// accepted: parentKeySubquery applies it correctly there, so nothing is
// dropped and there is nothing to reject. This is the negative side of the
// check — it fails if the rejection is scoped too broadly.
func TestPlanIsland_SameIslandWithFromWhereIsAccepted(t *testing.T) {
	edge := thenFromWhereEdge([]EdgeWhere{
		{On: "from", Column: "is_active", SQLValue: "true"},
	})

	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: a from-side condition within one island must be accepted, err = %v", err)
	}
	if len(plan.sections) != 1 {
		t.Fatalf("len(plan.sections) = %d, want 1", len(plan.sections))
	}
	if len(plan.thens) != 0 {
		t.Fatalf("len(plan.thens) = %d, want 0", len(plan.thens))
	}
}

// A to-side condition at a Then(...) boundary must be accepted: matchWhere
// ANDs "to" conditions in on every route, including the literal-keys
// branch, so nothing is dropped. This is the second negative side — it
// fails if the check looks at Where at all rather than at On == "from".
func TestPlanIsland_ThenBoundaryWithToWhereIsAccepted(t *testing.T) {
	edge := thenFromWhereEdge([]EdgeWhere{
		{On: "to", Column: "status", SQLValue: "'active'"},
	})

	plan, err := planIsland([]Node{Then(NewNode(edge))})
	if err != nil {
		t.Fatalf("planIsland: a to-side condition at a Then boundary must be accepted, err = %v", err)
	}
	if len(plan.thens) != 1 {
		t.Fatalf("len(plan.thens) = %d, want 1", len(plan.thens))
	}
	if len(plan.sections) != 0 {
		t.Fatalf("len(plan.sections) = %d, want 0", len(plan.sections))
	}
}

// A from-side condition mixed with a to-side condition on the SAME edge
// must still be rejected at a Then(...) boundary: the loop over
// edgeWhere() must not stop at the first entry or otherwise miss a "from"
// entry that isn't first in the slice.
func TestPlanIsland_ThenBoundaryWithMixedFromAndToWhereIsRejected(t *testing.T) {
	edge := thenFromWhereEdge([]EdgeWhere{
		{On: "to", Column: "status", SQLValue: "'active'"},
		{On: "from", Column: "is_active", SQLValue: "true"},
	})

	_, err := planIsland([]Node{Then(NewNode(edge))})
	var want *ThenBoundaryFromWhereError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ThenBoundaryFromWhereError", err)
	}
}

// --- multi-hop scoping (Sides[0] only) -----------------

// TestPlanIsland_ThenBoundaryMultiHopSide0FromWhereIsRejected: a
// multi-hop edge's Sides[0] (the hop matched against the ISLAND's
// parent) carries a from-side condition — the exact same failure mode
// thenFromWhereEdge's single-hop fixture already covers, now pinned for a
// multi-hop edge too: edgeWhere() (node.go) always resolves to
// sides()[0].Where, so this must reject identically.
func TestPlanIsland_ThenBoundaryMultiHopSide0FromWhereIsRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.Sides[0].Where = []EdgeWhere{
		{On: "from", Column: "is_active", SQLValue: "true"},
	}

	_, err := planIsland([]Node{Then(NewNode(edge))})
	var want *ThenBoundaryFromWhereError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ThenBoundaryFromWhereError", err)
	}
	if want.RelName != "Leaves" {
		t.Fatalf("RelName = %q, want %q", want.RelName, "Leaves")
	}
}

// TestPlanIsland_ThenBoundaryMultiHopSide1FromWhereIsAccepted: a
// multi-hop edge's Sides[1] (an INTERMEDIATE hop, not the one matched
// against the island's parent) carries a from-side condition. Unlike
// Sides[0]'s, this condition is rendered inside that hop's own JOIN ON
// clause (edgeJoinChain, compose.go) — a Then boundary's route-4 parent
// query builds that JOIN chain identically to routes 1/2, so nothing
// about it is ever dropped by crossing the boundary. planIsland must
// therefore accept the tree, and the resulting route-4 SQL must actually
// contain the condition inside the JOIN's ON.
func TestPlanIsland_ThenBoundaryMultiHopSide1FromWhereIsAccepted(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.Sides[1].Where = []EdgeWhere{
		{On: "from", Column: "mid_status", SQLValue: "'ready'"},
	}

	plan, err := planIsland([]Node{Then(NewNode(edge))})
	if err != nil {
		t.Fatalf("planIsland: a from-side condition on Sides[1] must be accepted at a Then boundary, err = %v", err)
	}
	if len(plan.thens) != 1 {
		t.Fatalf("len(plan.thens) = %d, want 1", len(plan.thens))
	}
	if len(plan.sections) != 0 {
		t.Fatalf("len(plan.sections) = %d, want 0", len(plan.sections))
	}

	// Render the route-4 SQL the same way hooks.go's runThen builds it
	// (childRoot.scanCols() + edgeJoinChain), and confirm Sides[1]'s own
	// from-side condition survives into the JOIN's ON clause.
	childRoot := edgeRootParts{edge.child(), edge}
	joins, carried, matchAlias := edgeJoinChain(edge)
	q, err := buildEdgeSelect(edgeSelectSpec{
		edge:            edge,
		cols:            childRoot.scanCols(),
		from:            tableExpr(childRoot.schemaName(), childRoot.tableName()),
		alias:           srcAlias,
		joins:           joins,
		carried:         carried,
		matchAlias:      matchAlias,
		keys:            [][]any{{1}},
		collapseOrderBy: true,
	})
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	if !strings.Contains(sql, `"_sl_h1"."mid_status" = 'ready'`) {
		t.Fatalf("route-4 SQL must carry Sides[1]'s from-side condition in the JOIN ON:\n%s", sql)
	}
	if !strings.Contains(sql, "INNER JOIN") {
		t.Fatalf("route-4 SQL must still contain the multi-hop JOIN:\n%s", sql)
	}
}

// TestPlanIsland_NestedThenBoundaryWithFromWhereIsRejectedOnlyWhenReached
// pins the actual scope of the fail-closed guarantee documented on
// ThenBoundaryFromWhereError: planIsland only walks the nodes it is
// literally called with. A Then(...) boundary nested inside ANOTHER
// Then(...)'s own children is not walked by the top-level call — it is
// only reached by the second planIsland call runThen (hooks.go) makes
// against plan.thens[i].children once that outer boundary actually runs.
// So a from-side violation on the INNER boundary is invisible to the
// top-level call and is only caught once its parent island's children are
// planned on their own (exactly as All's runThen does at runtime; Build
// never calls runThen, so it never reaches this second call at all).
func TestPlanIsland_NestedThenBoundaryWithFromWhereIsRejectedOnlyWhenReached(t *testing.T) {
	inner := thenFromWhereEdge([]EdgeWhere{
		{On: "from", Column: "is_active", SQLValue: "true"},
	})
	outer := thenFromWhereEdge(nil)

	tree := []Node{Then(NewNode(outer, Then(NewNode(inner))))}

	// The top-level call only sees the outer Then boundary; the inner one
	// is stashed as unwalked children, so no error surfaces here.
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("top-level planIsland: the inner Then boundary is not walked yet, so err must be nil, got %v", err)
	}
	if len(plan.thens) != 1 {
		t.Fatalf("len(plan.thens) = %d, want 1", len(plan.thens))
	}

	// Only planning the outer boundary's own children (what runThen does
	// once it actually reaches this boundary) walks into the inner one and
	// catches the violation.
	_, err = planIsland(plan.thens[0].children)
	var want *ThenBoundaryFromWhereError
	if !errors.As(err, &want) {
		t.Fatalf("nested planIsland: err = %v, want *ThenBoundaryFromWhereError", err)
	}
}
