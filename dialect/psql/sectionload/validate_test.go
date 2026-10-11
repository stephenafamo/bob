package sectionload

import (
	"errors"
	"strings"
	"testing"
)

// Fixtures for validatePlan: a three-level tree (vpParent -> vpChild ->
// vpGrandchild) so the "child of a restricted edge" checks have a real
// nested edge to exercise. validatePlan never scans or builds a model, so
// TableParts.NewBuf is left nil throughout — it is never called.

type (
	vpParent struct{ ID int }
	vpChild  struct {
		ID       int
		ParentID int
	}
)

type vpGrandchild struct {
	ID      int
	ChildID int
}

func vpChildParts(cols ...Column) TableParts[vpChild] {
	return TableParts[vpChild]{Table: "vp_children", Columns: cols}
}

func vpGrandchildParts(cols ...Column) TableParts[vpGrandchild] {
	return TableParts[vpGrandchild]{Table: "vp_grandchildren", Columns: cols}
}

func TestValidatePlan_UnrestrictedEdgeNeedsNoFromBuf(t *testing.T) {
	e := Edge[vpParent, vpChild]{
		RelName: "Children",
		Child:   vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		// ColumnPlan zero value: unrestricted. ParentKeyFromBuf/
		// ChildKeyFromBuf intentionally left nil.
	}
	if err := validatePlan([]Node{NewNode(e)}, nil); err != nil {
		t.Fatalf("validatePlan = %v, want nil (an unrestricted edge's fallback scans every column, so it needs no *FromBuf)", err)
	}
}

func TestValidatePlan_RestrictedEdgeMissingChildKeyFromBufIsError(t *testing.T) {
	e := Edge[vpParent, vpChild]{
		RelName:    "Children",
		Child:      vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan: ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "parent_id"}}},
		// ChildKeyFromBuf intentionally nil: this edge's own childKeyFromBufAny
		// would fall back to BuildFor(nil) on a buffer that never scanned "secret".
	}
	err := validatePlan([]Node{NewNode(e)}, nil)
	var keyErr *MissingKeyFromBufError
	if !errors.As(err, &keyErr) || keyErr.RelName != "Children" || keyErr.Field != "ChildKeyFromBuf" {
		t.Fatalf("err = %v, want *MissingKeyFromBufError{RelName: \"Children\", Field: \"ChildKeyFromBuf\"}", err)
	}
}

func TestValidatePlan_RestrictedEdgeChildMissingParentKeyFromBufIsError(t *testing.T) {
	child := Edge[vpParent, vpChild]{
		RelName:         "Children",
		Child:           vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan:      ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "parent_id"}}},
		ChildKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}
	grandchild := Edge[vpChild, vpGrandchild]{
		RelName:  "Grandchildren",
		FromCols: []string{"id"},
		Child:    vpGrandchildParts(Column{Name: "id"}, Column{Name: "child_id"}),
		// ParentKeyFromBuf intentionally nil: parentKeyFromBufAny would be
		// called against `child`'s partially-scanned buffer (missing "secret"),
		// falling back to BuildFor(nil) on it.
	}
	err := validatePlan([]Node{NewNode(child, NewNode(grandchild))}, nil)
	var keyErr *MissingKeyFromBufError
	if !errors.As(err, &keyErr) || keyErr.RelName != "Grandchildren" || keyErr.Field != "ParentKeyFromBuf" {
		t.Fatalf("err = %v, want *MissingKeyFromBufError{RelName: \"Grandchildren\", Field: \"ParentKeyFromBuf\"}", err)
	}
	// The restricted buffer being read against belongs to the PARENT edge
	// ("Children"), not the erroring child ("Grandchildren") — the buffer
	// owner and the edge missing the override are different relations here.
	if keyErr.RestrictedRelName != "Children" {
		t.Fatalf("err = %v, want RestrictedRelName == \"Children\" (the parent edge owns the restricted scan buffer)", err)
	}
	msg := keyErr.Error()
	if !strings.Contains(msg, "Grandchildren") || !strings.Contains(msg, "Children") {
		t.Fatalf("error message must name both the erroring relation and the restricted relation, got: %s", msg)
	}
}

func TestValidatePlan_RestrictedEdgeChildMissingRequiredColumnIsError(t *testing.T) {
	child := Edge[vpParent, vpChild]{
		RelName: "Children",
		Child:   vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		// "id" only: the grandchild's FromCols ("parent_id") is not fetched.
		ColumnPlan:      ColumnPlan{Model: []Column{{Name: "id"}}},
		ChildKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}
	grandchild := Edge[vpChild, vpGrandchild]{
		RelName:          "Grandchildren",
		FromCols:         []string{"parent_id"},
		Child:            vpGrandchildParts(Column{Name: "id"}, Column{Name: "child_id"}),
		ParentKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}
	err := validatePlan([]Node{NewNode(child, NewNode(grandchild))}, nil)
	var colErr *MissingRequiredColumnError
	if !errors.As(err, &colErr) || colErr.RelName != "Children" || colErr.ChildRelName != "Grandchildren" || colErr.Column != "parent_id" {
		t.Fatalf("err = %v, want *MissingRequiredColumnError{RelName: \"Children\", ChildRelName: \"Grandchildren\", Column: \"parent_id\"}", err)
	}
}

func TestValidatePlan_RestrictedEdgeFullyWiredIsOK(t *testing.T) {
	child := Edge[vpParent, vpChild]{
		RelName:         "Children",
		Child:           vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan:      ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "parent_id"}}},
		ChildKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}
	grandchild := Edge[vpChild, vpGrandchild]{
		RelName:          "Grandchildren",
		FromCols:         []string{"parent_id"},
		Child:            vpGrandchildParts(Column{Name: "id"}, Column{Name: "child_id"}),
		ParentKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}
	if err := validatePlan([]Node{NewNode(child, NewNode(grandchild))}, nil); err != nil {
		t.Fatalf("validatePlan = %v, want nil (both *FromBuf overrides set, required column fetched)", err)
	}
}

func TestValidatePlan_ThenBoundaryEdgeAlsoRequiresChildKeyFromBuf(t *testing.T) {
	e := Edge[vpParent, vpChild]{
		RelName:    "Children",
		Child:      vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan: ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "parent_id"}}},
		// ChildKeyFromBuf nil: a Then(...) boundary's child becomes a nested
		// island's root (edgeRootParts), and execIsland's rootChildKeyFn is
		// literally e.childKeyFromBufAny — the same fallback risk as an
		// in-island section, just reached through runThen instead.
	}
	err := validatePlan([]Node{Then(NewNode(e))}, nil)
	var keyErr *MissingKeyFromBufError
	if !errors.As(err, &keyErr) || keyErr.RelName != "Children" || keyErr.Field != "ChildKeyFromBuf" {
		t.Fatalf("err = %v, want *MissingKeyFromBufError{RelName: \"Children\", Field: \"ChildKeyFromBuf\"} (Then wrapping must not bypass the check)", err)
	}
}

func TestValidatePlan_RestrictedEdgeMissingOwnToColumnIsError(t *testing.T) {
	// A restricted edge's OWN ChildKeyFromBuf reads
	// its own ToCols — dropping one from the ColumnPlan must be a pre-flight
	// error, not a silent zero-value stitch failure.
	e := Edge[vpParent, vpChild]{
		RelName:         "Children",
		ToCols:          []string{"parent_id"}, // Required by this edge's ChildKeyFromBuf
		Child:           vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan:      ColumnPlan{Model: []Column{{Name: "id"}}}, // does not include ToCols ("parent_id")
		ChildKeyFromBuf: func(Buf[vpChild]) []any { return nil },   // non-nil override
	}

	err := validatePlan([]Node{NewNode(e)}, nil)
	var want *MissingRequiredColumnError
	if !errors.As(err, &want) {
		t.Fatalf("expected MissingRequiredColumnError, got %v", err)
	}
	if want.RelName != "Children" || want.ChildRelName != "Children" || want.Column != "parent_id" {
		t.Fatalf("error must name the edge itself on both sides: %+v", want)
	}
}

func TestValidatePlan_ChecksEverySiblingIncludingThen(t *testing.T) {
	// Place two restricted sibling edges under the same parent: the first
	// keeps its ToCols in the ColumnPlan, and only the second (the one
	// wrapped in Then) drops its ToCols from the ColumnPlan. The second must
	// fail because of its own missing ToCols.
	parent := Edge[vpParent, vpChild]{
		RelName:         "Children",
		Child:           vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan:      ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "parent_id"}}},
		ChildKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}

	// First sibling (restricted child): ToCols matches ColumnPlan → OK
	child1 := Edge[vpChild, vpGrandchild]{
		RelName:          "Grandchildren1",
		ToCols:           []string{"id"},
		Child:            vpGrandchildParts(Column{Name: "id"}, Column{Name: "child_id"}),
		ParentKeyFromBuf: func(Buf[vpChild]) []any { return nil },
		ChildKeyFromBuf:  func(Buf[vpGrandchild]) []any { return nil },
		ColumnPlan:       ColumnPlan{Model: []Column{{Name: "id"}}}, // ToCols matches
	}

	// Second sibling (Then boundary, restricted child): ToCols NOT in ColumnPlan → ERROR
	child2 := Edge[vpChild, vpGrandchild]{
		RelName:          "Grandchildren2",
		ToCols:           []string{"parent_id"}, // Requires parent_id
		Child:            vpGrandchildParts(Column{Name: "id"}, Column{Name: "child_id"}),
		ParentKeyFromBuf: func(Buf[vpChild]) []any { return nil },
		ChildKeyFromBuf:  func(Buf[vpGrandchild]) []any { return nil },
		ColumnPlan:       ColumnPlan{Model: []Column{{Name: "id"}}}, // parent_id is NOT in ColumnPlan
	}

	err := validatePlan([]Node{NewNode(parent, NewNode(child1), Then(NewNode(child2)))}, nil)
	var colErr *MissingRequiredColumnError
	if !errors.As(err, &colErr) || colErr.RelName != "Grandchildren2" || colErr.ChildRelName != "Grandchildren2" || colErr.Column != "parent_id" {
		t.Fatalf("err = %v, want *MissingRequiredColumnError{RelName: \"Grandchildren2\", ChildRelName: \"Grandchildren2\", Column: \"parent_id\"}", err)
	}
}

func TestValidatePlan_ChecksTwoNestedThenLevels(t *testing.T) {
	// Place a restricted edge in each of two or more nested Then levels, with
	// only the deepest edge dropping its own ToCols from the ColumnPlan. The
	// deepest edge must fail because of its own missing ToCols.
	type vpGreatGrandchild struct {
		ID           int
		GrandchildID int
	}

	parent := Edge[vpParent, vpChild]{
		RelName:         "Children",
		Child:           vpChildParts(Column{Name: "id"}, Column{Name: "parent_id"}, Column{Name: "secret"}),
		ColumnPlan:      ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "parent_id"}}},
		ChildKeyFromBuf: func(Buf[vpChild]) []any { return nil },
	}

	// First Then level: restricted child, ToCols matches ColumnPlan → OK
	child := Edge[vpChild, vpGrandchild]{
		RelName:          "Grandchildren",
		ToCols:           []string{"id"},
		Child:            vpGrandchildParts(Column{Name: "id"}, Column{Name: "child_id"}, Column{Name: "secret2"}),
		ColumnPlan:       ColumnPlan{Model: []Column{{Name: "id"}}, Internal: []Column{{Name: "child_id"}}},
		ParentKeyFromBuf: func(Buf[vpChild]) []any { return nil },
		ChildKeyFromBuf:  func(Buf[vpGrandchild]) []any { return nil },
	}

	greatGrandchildParts := func(cols ...Column) TableParts[vpGreatGrandchild] {
		return TableParts[vpGreatGrandchild]{Table: "vp_great_grandchildren", Columns: cols}
	}

	// Second Then level (deepest): restricted child, ToCols NOT in ColumnPlan → ERROR
	greatGrandchild := Edge[vpGrandchild, vpGreatGrandchild]{
		RelName:          "GreatGrandchildren",
		ToCols:           []string{"child_id"}, // Requires child_id
		Child:            greatGrandchildParts(Column{Name: "id"}, Column{Name: "grandchild_id"}),
		ColumnPlan:       ColumnPlan{Model: []Column{{Name: "id"}}}, // child_id is NOT in ColumnPlan
		ParentKeyFromBuf: func(Buf[vpGrandchild]) []any { return nil },
		ChildKeyFromBuf:  func(Buf[vpGreatGrandchild]) []any { return nil },
	}

	err := validatePlan([]Node{
		NewNode(parent, Then(NewNode(child, Then(NewNode(greatGrandchild))))),
	}, nil)

	var colErr *MissingRequiredColumnError
	if !errors.As(err, &colErr) || colErr.RelName != "GreatGrandchildren" || colErr.ChildRelName != "GreatGrandchildren" || colErr.Column != "child_id" {
		t.Fatalf("err = %v, want *MissingRequiredColumnError{RelName: \"GreatGrandchildren\", ChildRelName: \"GreatGrandchildren\", Column: \"child_id\"}", err)
	}
}
