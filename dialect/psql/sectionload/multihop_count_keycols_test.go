package sectionload

// Negative tests for countKeyCols (node.go), which must check the column
// names and order of KeyCols, not just their count. For example, with
// Sides[0].ToCols=["parent_id"] and a mixed-up KeyCols=[{Name:"leaf_id"}],
// a length-only check (1==1) would pass, SQL would be built with the wrong
// column name, and counts could be silently assigned to the wrong parents (or
// every parent could get zero).

import (
	"errors"
	"testing"
)

// multiHopTwoHopCompositeCountEdge: the CountEdge counterpart of
// multiHopTwoHopCompositeEdge (multihop_sql_test.go) — mh_composite_parents(id1,id2)
// -> mh_mid -> mh_leaves, a 2-column composite key, for
// TestMultiHopCountCountKeyCols_CompositeOrderMismatchIsRejected.
func multiHopTwoHopCompositeCountEdge() CountEdge[multiHopCompositeParent, multiHopLeaf] {
	return CountEdge[multiHopCompositeParent, multiHopLeaf]{
		RelName: "CompositeLeavesCount",
		Sides: []Side{
			{
				FromTable: "mh_composite_parents", FromCols: []string{"id1", "id2"},
				ToTable: "mh_mid", ToCols: []string{"parent_id1", "parent_id2"},
			},
			{FromTable: "mh_mid", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
		},
		Child: multiHopLeafParts(),
		KeyCols: []Column{
			{Name: "parent_id1", DBType: "int4"},
			{Name: "parent_id2", DBType: "int4"},
		},
		NewCarriedBuf: func() Buf[multiHopCompositeParent] { return &multiHopCompositeParentBuf{} },
		ParentKey:     func(p multiHopCompositeParent) []any { return []any{p.ID1, p.ID2} },
		ChildKey:      func(c multiHopLeaf) []any { return []any{c.ID} },
		SetCount:      func(p multiHopCompositeParent, n int64) error { return nil },
	}
}

// TestMultiHopCountCountKeyCols_NameMismatchIsRejected: when KeyCols[0].Name differs
// from Sides[0].ToCols[0] (the lengths match), countKeyCols (node.go, via
// planIsland) must return CountKeyColsMismatchError.
func TestMultiHopCountCountKeyCols_NameMismatchIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	if got, want := edge.Sides[0].ToCols[0], "parent_id"; got != want {
		t.Fatalf("fixture precondition failed: Sides[0].ToCols[0] = %q, want %q", got, want)
	}
	edge.KeyCols[0].Name = "leaf_id" // Sides[0].ToCols[0] is "parent_id" — a real column on mh_mid, but the WRONG one.

	_, err := planIsland([]Node{NewCountNode(edge)})
	var want *CountKeyColsMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountKeyColsMismatchError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestMultiHopCountCountKeyCols_CompositeOrderMismatchIsRejected: for a composite key
// (2 columns) whose KeyCols has the same number of columns but the opposite
// order to Sides[0].ToCols (parent_id2, parent_id1), the same
// CountKeyColsMismatchError must be returned. A count-only check would not
// detect this case.
func TestMultiHopCountCountKeyCols_CompositeOrderMismatchIsRejected(t *testing.T) {
	edge := multiHopTwoHopCompositeCountEdge()
	if got, want := edge.Sides[0].ToCols, ([]string{"parent_id1", "parent_id2"}); got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("fixture precondition failed: Sides[0].ToCols = %v, want %v", got, want)
	}
	edge.KeyCols[0], edge.KeyCols[1] = edge.KeyCols[1], edge.KeyCols[0] // reversed order, same length (2==2).

	_, err := planIsland([]Node{NewCountNode(edge)})
	var want *CountKeyColsMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountKeyColsMismatchError", err)
	}
	if want.RelName != "CompositeLeavesCount" {
		t.Fatalf("err = %+v, want RelName=CompositeLeavesCount", want)
	}
}
