package sectionload

// Unit tests for resolveColumnPlans/resolveEdgeColumnPlan: error cases and
// non-mutation of the caller's nodes. fakeParent/fakeChild/fakeChildBuf are
// the shared fixtures defined in columnplan_test.go.

import (
	"errors"
	"testing"
)

// colRestrictFakeEdge is a valid, single-hop Edge[*fakeParent, *fakeChild] fixture
// for resolveColumnPlans/resolveEdgeColumnPlan tests: it has everything
// those functions read (sides()/toCols()/child()/relName()/options()), but
// no ChildKey/Stitch — tests here never reach compose/exec, only the
// resolution step (validatePlan/composeIsland are exercised by the
// compose-level tests instead).
func colRestrictFakeEdge() Edge[*fakeParent, *fakeChild] {
	return Edge[*fakeParent, *fakeChild]{
		RelName:  "FakeChild",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*fakeChild]{
			Table: "fake_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "secret", DBType: "text"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*fakeChild] { return &fakeChildBuf{} },
		},
		ParentKey: func(p *fakeParent) []any { return []any{p.ID} },
		ChildKey:  func(c *fakeChild) []any { return []any{0} },
	}
}

// colRestrictFakeCountEdge is the count-node counterpart of colRestrictFakeEdge, for the
// count-node restriction test (CountColumnRestrictionUnsupportedError).
func colRestrictFakeCountEdge() CountEdge[*fakeParent, *fakeChild] {
	return CountEdge[*fakeParent, *fakeChild]{
		RelName:  "FakeChildCount",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*fakeChild]{
			Table:   "fake_children",
			Columns: []Column{{Name: "id", DBType: "int4"}, {Name: "parent_id", DBType: "int4"}},
			NewBuf:  func() Buf[*fakeChild] { return &fakeChildBuf{} },
		},
		ParentKey: func(p *fakeParent) []any { return []any{p.ID} },
		ChildKey:  func(c *fakeChild) []any { return []any{0} },
		SetCount:  func(p *fakeParent, n int64) error { return nil },
	}
}

// --- resolveColumnPlans error cases -----------------------------

// Both Only and Except given: *ColumnSelectionConflictError.
func TestResolveColumnPlans_BothOnlyAndExceptIsConflict(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = SelectOptions{}.Only("id").Except("secret")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *ColumnSelectionConflictError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ColumnSelectionConflictError", err)
	}
}

// Only() called with an empty list: *ColumnSelectionEmptyError{Field:"Only"}.
func TestResolveColumnPlans_EmptyOnlyIsError(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = Only()

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *ColumnSelectionEmptyError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ColumnSelectionEmptyError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Only" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Only", want)
	}
}

// Except() called with an empty list: Field:"Except".
func TestResolveColumnPlans_EmptyExceptIsError(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = Except()

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *ColumnSelectionEmptyError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ColumnSelectionEmptyError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Except" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Except", want)
	}
}

// A column name that does not exist: *UnknownColumnError.
func TestResolveColumnPlans_UnknownColumnIsError(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = Only("does_not_exist")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *UnknownColumnError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *UnknownColumnError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Only" || want.Column != "does_not_exist" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Only Column=does_not_exist", want)
	}
}

// A join key column given to Except: *KeyColumnExcludedError (raised via
// NewColumnPlan).
func TestResolveColumnPlans_ExceptExcludingKeyIsError(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = Except("parent_id")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *KeyColumnExcludedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *KeyColumnExcludedError", err)
	}
	if want.Column != "parent_id" {
		t.Fatalf("err.Column = %q, want parent_id", want.Column)
	}
}

// Only/Except on a count node: *CountColumnRestrictionUnsupportedError.
func TestResolveColumnPlans_CountNodeWithOnlyIsError(t *testing.T) {
	c := colRestrictFakeCountEdge()
	c.Options = Only("id")

	_, err := resolveColumnPlans([]Node{NewCountNode(c)})
	var want *CountColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountColumnRestrictionUnsupportedError", err)
	}
	if want.RelName != "FakeChildCount" {
		t.Fatalf("err.RelName = %q, want FakeChildCount", want.RelName)
	}
}

// An empty SelectOptions{} leaves ColumnPlan at its zero value (the SQL
// stays byte-identical; SQL generation itself is covered by the compose-level
// tests, so this only checks that the plan resolves to the zero value).
//
// The check compares against nil rather than len==0: a nil slice means
// "unrestricted" while a non-nil empty slice means "restricted to zero
// public columns", and a length-only check would accept the latter.
func TestResolveColumnPlans_NoRestrictionLeavesColumnPlanZero(t *testing.T) {
	e := colRestrictFakeEdge() // Options is the zero value (no Mods, no Only/Except)

	resolved, err := resolveColumnPlans([]Node{NewNode(e)})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	got := resolved[0].edge.(Edge[*fakeParent, *fakeChild]).ColumnPlan
	if got.Model != nil || got.Internal != nil {
		t.Fatalf("ColumnPlan = %+v, want exact zero value", got)
	}
}

// The same column named twice on one side: *DuplicateColumnError.
func TestResolveColumnPlans_DuplicateColumnNameIsError(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = Except("secret", "secret")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *DuplicateColumnError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *DuplicateColumnError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Except" || want.Column != "secret" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Except Column=secret", want)
	}
}

// Only/Except on a multi-hop edge: *MultiHopColumnRestrictionUnsupportedError.
// multiHopTwoHopEdge (multihop_sql_test.go) is a real 2-side (len(Sides)==2) fixture.
func TestResolveColumnPlans_MultiHopEdgeIsError(t *testing.T) {
	e := multiHopTwoHopEdge()
	e.Options = Only("label")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *MultiHopColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *MultiHopColumnRestrictionUnsupportedError", err)
	}
	if want.RelName != "Leaves" {
		t.Fatalf("err.RelName = %q, want Leaves", want.RelName)
	}
}

// --- resolveColumnPlans does not mutate its input ----------------------------

// TestResolveColumnPlans_DoesNotMutateOriginalEdge checks that
// resolveColumnPlans/withColumnPlan set ColumnPlan only on a copy of the
// edge. The caller's original Node/Edge must keep a zero ColumnPlan, which is
// what makes it safe to pass the same Node value to All()/Build() repeatedly.
//
// Node.edge is an edgeSpec interface with no ColumnPlan getter (only the
// columnPlanRestricts() bool), so the original is inspected by asserting the
// concrete type.
func TestResolveColumnPlans_DoesNotMutateOriginalEdge(t *testing.T) {
	e := colRestrictFakeEdge()
	e.Options = Except("secret")
	original := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{original})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	// The original Node value (still held by the caller) keeps a zero
	// ColumnPlan.
	origPlan := original.edge.(Edge[*fakeParent, *fakeChild]).ColumnPlan
	if origPlan.Model != nil || origPlan.Internal != nil {
		t.Fatalf("original edge's ColumnPlan = %+v, want exact zero value (must not be mutated)", origPlan)
	}

	// The resolved Node's ColumnPlan is non-zero (everything but secret is in Model).
	resolvedPlan := resolved[0].edge.(Edge[*fakeParent, *fakeChild]).ColumnPlan
	if len(resolvedPlan.Model) != 2 {
		t.Fatalf("resolved edge's ColumnPlan.Model = %v, want 2 columns (id, parent_id)", resolvedPlan.Model)
	}
}

// TestResolveColumnPlans_DoesNotMutateOriginalChildEdge is the child-node
// version of the test above. That test only inspects the top-level edge, so
// it would not notice resolveNode in columnselect.go writing into the
// original child edges: resolveNode copies the child slice (n.children)
// before rewriting it, and removing that copy would only corrupt the
// original tree's child edges.
func TestResolveColumnPlans_DoesNotMutateOriginalChildEdge(t *testing.T) {
	child := Edge[*fakeChild, *fakeChild]{
		RelName:  "NestedFakeChild",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Options:  Except("secret"),
		Child:    colRestrictFakeEdge().Child,
		ParentKey: func(*fakeChild) []any {
			return []any{0}
		},
		ChildKey: func(*fakeChild) []any {
			return []any{0}
		},
	}

	root := colRestrictFakeEdge()
	original := NewNode(root, NewNode(child))

	resolved, err := resolveColumnPlans([]Node{original})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	originalChildPlan := original.children[0].edge.(Edge[*fakeChild, *fakeChild]).ColumnPlan
	if originalChildPlan.Model != nil || originalChildPlan.Internal != nil {
		t.Fatalf(
			"original child plan = %+v, want exact zero value",
			originalChildPlan,
		)
	}

	resolvedChildPlan := resolved[0].children[0].edge.(Edge[*fakeChild, *fakeChild]).ColumnPlan
	if len(resolvedChildPlan.Model) == 0 {
		t.Fatalf("resolved child plan = %+v, want restricted plan", resolvedChildPlan)
	}
}
