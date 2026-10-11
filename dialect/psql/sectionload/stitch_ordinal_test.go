package sectionload

// stitchInto's ords parameter. A UNION ALL does not preserve any one
// branch's internal row order, so when an edge's own SelectOptions
// requested an ORDER BY, buildEdgeSelect captures it as a per-row ordinal;
// stitchInto is where that ordinal actually takes
// effect, by stable-sorting children before grouping them by parent key.

import (
	"database/sql"
	"reflect"
	"testing"
)

func stitchOrdinalEdge(record *[]int) Edge[builderCapParent, builderCapChild] {
	return Edge[builderCapParent, builderCapChild]{
		RelName:   "Child",
		FromCols:  []string{"id"},
		ToCols:    []string{"parent_id"},
		ToMany:    true,
		ParentKey: func(p builderCapParent) []any { return []any{p.ID} },
		ChildKey:  func(c builderCapChild) []any { return []any{c.ParentID} },
		Stitch: func(p builderCapParent, cs []builderCapChild) error {
			for _, c := range cs {
				*record = append(*record, c.ID)
			}
			return nil
		},
	}
}

func TestStitchInto_OrdinalSort(t *testing.T) {
	var stitched []int
	e := stitchOrdinalEdge(&stitched)

	parents := []any{builderCapParent{ID: 1}}
	// Scan order (as a UNION ALL might actually return it) does not match
	// the requested ORDER BY at all.
	children := []any{
		builderCapChild{ID: 3, ParentID: 1},
		builderCapChild{ID: 1, ParentID: 1},
		builderCapChild{ID: 2, ParentID: 1},
	}
	ords := []sql.NullInt64{
		{Int64: 3, Valid: true}, // row_number() for child ID 3 (3rd in the requested order)
		{Int64: 1, Valid: true}, // child ID 1 is 1st
		{Int64: 2, Valid: true}, // child ID 2 is 2nd
	}

	sorted, err := stitchInto(e, parents, children, nil, nil, ords)
	if err != nil {
		t.Fatalf("stitchInto: %v", err)
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(stitched, want) {
		t.Fatalf("stitched order: got %v, want %v (ordinal must be honored)", stitched, want)
	}
	// The returned slice must reflect the sort too: the caller (stitchIsland) writes this back
	// into items[n.set] so a nested Then(...) under this edge sees the same
	// order stitching used, not the original scan order.
	returnedOrder := make([]int, 0, len(sorted))
	for _, c := range sorted {
		returnedOrder = append(returnedOrder, c.(builderCapChild).ID)
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(returnedOrder, want) {
		t.Fatalf("returned children order: got %v, want %v", returnedOrder, want)
	}
}

// TestStitchInto_NoOrdinal_PreservesScanOrder is the negative side: when no
// ordinal was captured (the common case — the edge requested no ORDER BY),
// stitchInto must be a complete no-op on ordering, byte-identical to
// before this parameter existed.
func TestStitchInto_NoOrdinal_PreservesScanOrder(t *testing.T) {
	var stitched []int
	e := stitchOrdinalEdge(&stitched)

	parents := []any{builderCapParent{ID: 1}}
	children := []any{
		builderCapChild{ID: 3, ParentID: 1},
		builderCapChild{ID: 1, ParentID: 1},
		builderCapChild{ID: 2, ParentID: 1},
	}

	sorted, err := stitchInto(e, parents, children, nil, nil, nil)
	if err != nil {
		t.Fatalf("stitchInto: %v", err)
	}
	if want := []int{3, 1, 2}; !reflect.DeepEqual(stitched, want) {
		t.Fatalf("stitched order: got %v, want %v (scan order must be untouched)", stitched, want)
	}
	if !reflect.DeepEqual(sorted, children) {
		t.Fatalf("returned slice must equal input unchanged when ords is nil: got %v, want %v", sorted, children)
	}
}

// TestStitchInto_OrdinalCountMismatchIsError guards sortIndexByOrds's own
// invariant: a length
// mismatch between ords and children is always a caller bug (execIsland
// always produces exactly one ords entry per row), never "no ORDER BY
// requested" — that case is len(ords) == 0, not len(ords) != len(children).
// Returning a nil idx with no error for both cases would silently skip the
// reorder either way.
func TestStitchInto_OrdinalCountMismatchIsError(t *testing.T) {
	var stitched []int
	e := stitchOrdinalEdge(&stitched)

	parents := []any{builderCapParent{ID: 1}}
	children := []any{
		builderCapChild{ID: 3, ParentID: 1},
		builderCapChild{ID: 1, ParentID: 1},
	}
	// 3 ordinals for 2 children.
	ords := []sql.NullInt64{
		{Int64: 1, Valid: true},
		{Int64: 2, Valid: true},
		{Int64: 3, Valid: true},
	}

	if _, err := stitchInto(e, parents, children, nil, nil, ords); err == nil {
		t.Fatal("stitchInto: want error for ordinal/children count mismatch, got nil")
	}
}
