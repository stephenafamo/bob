package sectionload

// checkDeclaredCountCollisions unit coverage.
// An end-to-end test against a real generated schema can only exercise one
// half of the check (two identical PreloadCount declarations on the same
// parent query), because bob's generated PreloadCount alias
// ("__count_" + RelName) rarely collides with a real column or section
// name. This file exercises the other half (a declared alias colliding with
// a fixed layout column) directly against a hand-written
// TableParts/islandPlan, where the column names are under the test's control.

import (
	"errors"
	"testing"
)

func TestCheckDeclaredCountCollisions_AgainstFixedRootColumn(t *testing.T) {
	root := partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema: "",
		Table:  "bld_capparents",
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
	root := partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema:  "",
		Table:   "bld_capparents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
	}}
	edge := builderCapEdge() // childCols(): id, parent_id (bld_children)
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
	root := partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema:  "",
		Table:   "bld_capparents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
	}}
	plan := islandPlan{
		declaredCounts: []declaredCountPreload{
			{RelName: "Comments", Alias: "__count_Comments"},
		},
	}
	if err := checkDeclaredCountCollisions(root, plan); err != nil {
		t.Fatalf("expected no collision, got %v", err)
	}
}
