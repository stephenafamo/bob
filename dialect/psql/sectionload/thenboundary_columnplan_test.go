package sectionload

import (
	"reflect"
	"strings"
	"testing"
)

// TestEdgeRootParts_ThenBoundaryRespectsColumnPlan is a regression test for
// a gap in the column-restriction feature: runThen (hooks.go) treats a Then(...) boundary's child as the "root" of
// its own recursive island, but was passing e.child() — a plain,
// edge-unaware partsSpec whose cols() always returns the child table's
// FULL column list — into composeIsland/execIsland, while thenChildQuery
// (in the same function) already builds the actual SQL SELECT from the
// edge-aware, possibly-restricted e.childCols(). Once an edge is both
// column-restricted and a Then boundary target, the outer SELECT ends up
// referencing columns the actual query never selected — a column that
// does not exist in Postgres.
//
// runThen therefore wraps e.child() in edgeRootParts, which routes
// cols()/modelCols() through the SAME edge's ColumnPlan. This test
// exercises compose.go/exec.go's column-list-consuming functions
// directly (no DB needed — they are pure functions over partsSpec) with
// a restricted edgeRootParts, proving the excluded column never appears.
func TestEdgeRootParts_ThenBoundaryRespectsColumnPlan(t *testing.T) {
	edge := Edge[*fakeParent, *fakeChild]{
		ToCols: []string{"parent_id"},
		Child: TableParts[*fakeChild]{
			Table: "fake_children",
			Columns: []Column{
				{Name: "id", DBType: "bigint"},
				{Name: "secret", DBType: "text"},
				{Name: "parent_id", DBType: "bigint"},
			},
			NewBuf: func() Buf[*fakeChild] { return &fakeChildBuf{} },
		},
		// "secret" is excluded from BOTH Model and Internal: it must not
		// reach the physical SQL at all, unlike an auto-included key.
		ColumnPlan: ColumnPlan{
			Model:    []Column{{Name: "id", DBType: "bigint"}},
			Internal: []Column{{Name: "parent_id", DBType: "bigint"}},
		},
	}
	root := edgeRootParts{edge.child(), edge}

	if got, want := root.cols(), edge.childCols(); !reflect.DeepEqual(got, want) {
		t.Fatalf("edgeRootParts.cols() = %v, want the same as edge.childCols() = %v", got, want)
	}

	got := expectedColumns(root, islandPlan{})
	want := []string{colSet, colOrd, "id", "parent_id"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expectedColumns = %v, want %v (must not include \"secret\": it is excluded from the edge's ColumnPlan entirely)", got, want)
	}

	if err := checkWidth(root, islandPlan{}, DefaultMaxColumns); err != nil {
		t.Fatalf("checkWidth: %v", err)
	}

	if list := section0List(root, islandPlan{}); strings.Contains(list, "secret") {
		t.Fatalf("section0List references the excluded column \"secret\": %s", list)
	}
}

// TestEdgeRootParts_UnrestrictedFallsBackToFullChildColumns confirms an
// edgeRootParts wrapping an unrestricted edge (the zero ColumnPlan — every
// generated Edge today) behaves exactly like a plain partsAdapter: full
// column list, nil modelCols (the BuildFor "everything" fast path).
func TestEdgeRootParts_UnrestrictedFallsBackToFullChildColumns(t *testing.T) {
	edge := Edge[*fakeParent, *fakeChild]{
		Child: TableParts[*fakeChild]{
			Table: "fake_children",
			Columns: []Column{
				{Name: "id", DBType: "bigint"},
				{Name: "secret", DBType: "text"},
			},
			NewBuf: func() Buf[*fakeChild] { return &fakeChildBuf{} },
		},
		// ColumnPlan left at the zero value.
	}
	root := edgeRootParts{edge.child(), edge}

	if got, want := root.cols(), edge.Child.Columns; !reflect.DeepEqual(got, want) {
		t.Fatalf("edgeRootParts.cols() = %v, want the child table's full column list %v", got, want)
	}
	if got := root.modelCols(); got != nil {
		t.Fatalf("edgeRootParts.modelCols() = %v, want nil (unrestricted -> BuildFor fast path)", got)
	}
}
