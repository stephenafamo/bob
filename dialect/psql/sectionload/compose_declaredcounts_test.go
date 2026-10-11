package sectionload

// P-13 (§8 #8, W-4): checkDeclaredCountCollisions unit coverage.
// pkg/dbsections' TestP13_DeclaredCountColumnConflict can only exercise
// the M-5 half (two identical PreloadCount declarations on the same
// parent query) against the real fvuls schema, because bob's generated
// PreloadCount alias ("__count_" + RelName) never happens to collide
// with any real column/section name in that schema. This file exercises
// the OTHER half (a declared alias colliding with a fixed layout column)
// directly against a hand-written TableParts/islandPlan, where the
// column names are under this test's own control.

import (
	"errors"
	"testing"
)

func TestCheckDeclaredCountCollisions_AgainstFixedRootColumn(t *testing.T) {
	root := partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema: "",
		Table:  "p11_capparents",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "region", DBType: "text"},
		},
	}}
	plan := islandPlan{
		declaredCounts: []declaredCountPreload{
			{RelName: "Whatever", Alias: "id"}, // collides with root.cols()'s own "id"
		},
	}

	err := checkDeclaredCountCollisions(root, plan)
	var confErr *DeclaredCountColumnConflictError
	if err == nil {
		t.Fatal("expected *DeclaredCountColumnConflictError, got nil")
	}
	if !errors.As(err, &confErr) {
		t.Fatalf("expected *DeclaredCountColumnConflictError, got %T: %v", err, err)
	}
	if confErr.Alias != "id" {
		t.Fatalf("Alias: got %q, want %q", confErr.Alias, "id")
	}
}

func TestCheckDeclaredCountCollisions_AgainstSectionColumn(t *testing.T) {
	root := partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema:  "",
		Table:   "p11_capparents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
	}}
	edge := p11CapEdge() // childCols(): id, parent_id (p11_children)
	plan := islandPlan{
		sections: []flatNode{{set: 1, parentSet: 0, edge: edge}},
		declaredCounts: []declaredCountPreload{
			{RelName: "Whatever", Alias: "parent_id"}, // collides with the section's own child column
		},
	}

	err := checkDeclaredCountCollisions(root, plan)
	var confErr *DeclaredCountColumnConflictError
	if err == nil {
		t.Fatal("expected *DeclaredCountColumnConflictError, got nil")
	}
	if !errors.As(err, &confErr) {
		t.Fatalf("expected *DeclaredCountColumnConflictError, got %T: %v", err, err)
	}
	if confErr.Alias != "parent_id" {
		t.Fatalf("Alias: got %q, want %q", confErr.Alias, "parent_id")
	}
}

func TestCheckDeclaredCountCollisions_NoCollisionPasses(t *testing.T) {
	root := partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema:  "",
		Table:   "p11_capparents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
	}}
	plan := islandPlan{
		declaredCounts: []declaredCountPreload{
			{RelName: "TaskComments", Alias: "__count_TaskComments"},
		},
	}
	if err := checkDeclaredCountCollisions(root, plan); err != nil {
		t.Fatalf("expected no collision, got %v", err)
	}
}
