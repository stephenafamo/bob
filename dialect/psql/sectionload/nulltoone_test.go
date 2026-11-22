package sectionload

// A to-one relation with no child, or with a nil parent key, must still be
// marked as loaded.
//
// stitchInto is a pure function that never touches SQL (parents/children are
// already in-memory any slices), so these are plain unit tests that call it
// directly, covering both the negative cases and regressions without a
// database.

import (
	"fmt"
	"testing"
)

// nullToOneLoaded / nullToOneRels / nullToOneParent / nullToOneChild are test-only models that mimic
// the generated models' R / R.Loaded / typed-nil Preload pattern.

type nullToOneLoaded struct {
	Owner bool
	Kids  bool
}

type nullToOneRels struct {
	Owner  *nullToOneChild
	Kids   []*nullToOneChild
	Loaded nullToOneLoaded
}

type nullToOneParent struct {
	ID      int
	OwnerID *int // nil == nullable FK not set
	R       nullToOneRels
}

type nullToOneChild struct {
	ID int
	// RefID is a separate FK-like field so the to-many edge's ChildKey can
	// match on something other than the child's own PK (which the to-one
	// edge's ChildKey uses). Without this, both edges would group by the
	// same field and the to-many "multiple children" fixture below would
	// never match its parent.
	RefID int
}

// nullToOnePreloadOwner mimics only the type-assertion boundary on any. The
// generated template and the real generated Preload are verified by tests
// against the generated code.
func nullToOnePreloadOwner(p *nullToOneParent, retrieved any) error {
	rel, ok := retrieved.(*nullToOneChild)
	if !ok {
		return fmt.Errorf("nullToOneParent cannot load %T as %q", retrieved, "Owner")
	}
	p.R.Owner = rel
	p.R.Loaded.Owner = true
	return nil
}

// nullToOneToOneEdge has the same shape as the generated to-one Stitch
// (templates/table/01_edges.go.tpl): with no child it passes a typed nil
// across the any boundary (nullToOnePreloadOwner), and R.Loaded is always true.
func nullToOneToOneEdge() Edge[*nullToOneParent, *nullToOneChild] {
	return Edge[*nullToOneParent, *nullToOneChild]{
		RelName: "Owner",
		ToMany:  false,
		ParentKey: func(p *nullToOneParent) []any {
			if p.OwnerID == nil {
				return nil
			}
			return []any{*p.OwnerID}
		},
		ChildKey: func(c *nullToOneChild) []any { return []any{c.ID} },
		Stitch: func(p *nullToOneParent, children []*nullToOneChild) error {
			var related *nullToOneChild
			if len(children) > 0 {
				related = children[0]
			}
			var retrieved any = related // typed nil, exactly like the generated Stitch
			return nullToOnePreloadOwner(p, retrieved)
		},
	}
}

// nullToOneToManyEdge has the same shape as the generated to-many Stitch: even
// with zero children it passes an empty slice and sets R.Loaded.
func nullToOneToManyEdge() Edge[*nullToOneParent, *nullToOneChild] {
	return Edge[*nullToOneParent, *nullToOneChild]{
		RelName: "Kids",
		ToMany:  true,
		ParentKey: func(p *nullToOneParent) []any {
			if p.OwnerID == nil {
				return nil
			}
			return []any{*p.OwnerID}
		},
		ChildKey: func(c *nullToOneChild) []any { return []any{c.RefID} },
		Stitch: func(p *nullToOneParent, children []*nullToOneChild) error {
			p.R.Kids = children
			p.R.Loaded.Kids = true
			return nil
		},
	}
}

// --- Negative case: an untyped nil is rejected at the any boundary ---
//
// Passing an untyped nil directly to nullToOnePreloadOwner (the same type
// assertion as the to-one branch of the real generated Preload) yields
// ok=false and an error, unlike a typed nil (*nullToOneChild)(nil). This shows
// that the tests would catch a generated template that broke by passing nil
// directly instead of going through var related *nullToOneChild.
func TestNullToOnePreloadOwner_UntypedNilIsRejected(t *testing.T) {
	p := &nullToOneParent{ID: 1}
	err := nullToOnePreloadOwner(p, nil) // untyped nil, not (*nullToOneChild)(nil)
	if err == nil {
		t.Fatalf("nullToOnePreloadOwner(p, nil) = nil error, want an error (untyped nil must fail the type assertion, exactly like the real generated Preload)")
	}
	if p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should stay false when the any-boundary rejects the value: got true")
	}
}

func TestNullToOnePreloadOwner_TypedNilIsAccepted(t *testing.T) {
	p := &nullToOneParent{ID: 1}
	var related *nullToOneChild // typed nil — exactly what the generated Stitch passes
	if err := nullToOnePreloadOwner(p, related); err != nil {
		t.Fatalf("nullToOnePreloadOwner(p, (*nullToOneChild)(nil)) = %v, want nil error (typed nil must pass the type assertion)", err)
	}
	if p.R.Owner != nil {
		t.Fatalf("R.Owner = %#v, want nil", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should be true once a typed nil is accepted: got false")
	}
}

// --- Negative case (a): to-one with a nil parent key (nullable FK not set) ---

func TestStitchIntoToOneNullKey(t *testing.T) {
	e := nullToOneToOneEdge()
	var calls int
	origStitch := e.Stitch
	e.Stitch = func(p *nullToOneParent, children []*nullToOneChild) error {
		calls++
		return origStitch(p, children)
	}
	parents := []any{&nullToOneParent{ID: 1, OwnerID: nil}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if p.R.Owner != nil {
		t.Fatalf("R.Owner should stay nil: got %#v", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should be true: got false")
	}
	if calls != 1 {
		t.Fatalf("Stitch should be called exactly once for to-one with nil key: got %d", calls)
	}
}

// --- Negative case (b): to-one whose parent key is non-nil but has no matching child ---

func TestStitchIntoToOneNonNullKeyNoChild(t *testing.T) {
	e := nullToOneToOneEdge()
	ownerID := 99 // a key that no child has
	parents := []any{&nullToOneParent{ID: 1, OwnerID: &ownerID}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if p.R.Owner != nil {
		t.Fatalf("R.Owner should stay nil: got %#v", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should be true: got false")
	}
}

// --- Regression: to-one with a matching child (existing behavior unchanged) ---

func TestStitchIntoToOneWithChild(t *testing.T) {
	e := nullToOneToOneEdge()
	ownerID := 5
	parents := []any{&nullToOneParent{ID: 1, OwnerID: &ownerID}}
	children := []any{&nullToOneChild{ID: 5}}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if p.R.Owner == nil || p.R.Owner.ID != 5 {
		t.Fatalf("R.Owner should be the child with id=5: got %#v", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should be true: got false")
	}
}

// --- Regression: to-many with zero children (path that already worked) ---

func TestStitchIntoToManyZeroChildren(t *testing.T) {
	e := nullToOneToManyEdge()
	ownerID := 1
	parents := []any{&nullToOneParent{ID: 1, OwnerID: &ownerID}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if len(p.R.Kids) != 0 {
		t.Fatalf("R.Kids should be empty: got %#v", p.R.Kids)
	}
	if !p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids should be true: got false")
	}
}

// --- Regression: to-many with several children (path that already worked) ---

func TestStitchIntoToManyMultipleChildren(t *testing.T) {
	e := nullToOneToManyEdge()
	ownerID := 1
	parents := []any{&nullToOneParent{ID: 1, OwnerID: &ownerID}}
	children := []any{&nullToOneChild{ID: 10, RefID: 1}, &nullToOneChild{ID: 11, RefID: 1}}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if len(p.R.Kids) != 2 {
		t.Fatalf("R.Kids should have 2 entries: got %d", len(p.R.Kids))
	}
	if !p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids should be true: got false")
	}
}

// --- Regression: a to-many with a nil parent key never calls Stitch ---
//
// For a to-many edge with a nil parent key, stitchInto does not call Stitch
// at all, so R.Kids/R.Loaded.Kids stay at their zero values (this preserves
// the pre-existing behavior, where to-many relations keep their old
// semantics). The expectation that a nil key still yields Loaded=true for a
// to-many would be wrong. A schema may have no ToMany edge whose ParentKey
// can return nil, in which case this path is only theoretical, but it is the
// correct general runtime contract.
func TestStitchIntoToManyNullKeySkipsStitchEntirely(t *testing.T) {
	e := nullToOneToManyEdge()
	var calls int
	origStitch := e.Stitch
	e.Stitch = func(p *nullToOneParent, children []*nullToOneChild) error {
		calls++
		return origStitch(p, children)
	}
	parents := []any{&nullToOneParent{ID: 1, OwnerID: nil}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if p.R.Kids != nil {
		t.Fatalf("R.Kids should stay at the zero value (nil) because Stitch is never called: got %#v", p.R.Kids)
	}
	if p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids should stay false (a to-many with a nil key never calls Stitch): got true")
	}
	if calls != 0 {
		t.Fatalf("Stitch should not be called for to-many with nil key: got %d", calls)
	}
}

// --- Composite key matching ---
//
// mapKey packs a composite key into an [N]any array, so keys match only when
// every component matches. A partial match (only some components equal)
// falls into a different group.

// nullToOneToManyCompositeKeyEdge is a to-many Edge with a two-component composite
// key. Both the parent key and the child key return two-element slices,
// which mapKey packs into [2]any; only a full match lands in the same group.
func nullToOneToManyCompositeKeyEdge() Edge[*nullToOneParent, *nullToOneChild] {
	return Edge[*nullToOneParent, *nullToOneChild]{
		RelName: "Kids",
		ToMany:  true,
		ParentKey: func(p *nullToOneParent) []any {
			if p.OwnerID == nil {
				return nil
			}
			// composite key: [parent ID, parent's OwnerID]
			return []any{p.ID, *p.OwnerID}
		},
		ChildKey: func(c *nullToOneChild) []any {
			// composite key: [child ID, child RefID]
			return []any{c.ID, c.RefID}
		},
		Stitch: func(p *nullToOneParent, children []*nullToOneChild) error {
			p.R.Kids = children
			p.R.Loaded.Kids = true
			return nil
		},
	}
}

// TestStitchIntoToManyCompositeKeyNoMatch checks that a partial match on a
// composite key does not match. Parent key [1,7] and child key [2,7] differ
// in the first component, so they land in different groups and the child
// is not delivered.
func TestStitchIntoToManyCompositeKeyNoMatch(t *testing.T) {
	e := nullToOneToManyCompositeKeyEdge()
	var calls int
	origStitch := e.Stitch
	e.Stitch = func(p *nullToOneParent, children []*nullToOneChild) error {
		calls++
		return origStitch(p, children)
	}

	parentID := 7
	parents := []any{&nullToOneParent{ID: 1, OwnerID: &parentID}}
	children := []any{&nullToOneChild{ID: 2, RefID: 7}}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*nullToOneParent)
	if len(p.R.Kids) != 0 {
		t.Fatalf("R.Kids should be empty (a partially matching key is not delivered): got %#v", p.R.Kids)
	}
	if !p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids should be true (a to-many always calls Stitch): got false")
	}
	if calls != 1 {
		t.Fatalf("Stitch should be called exactly once: got %d", calls)
	}
}

// TestStitchIntoCompositeKeyMatchesAllComponents checks that a composite key
// matches only when all components match, and a partial match falls into a
// different group. Parent 1 [1,7] receives child 1 [1,7] but not child 3
// [1,8]; parent 2 [2,7] receives child 2 [2,7].
func TestStitchIntoCompositeKeyMatchesAllComponents(t *testing.T) {
	e := nullToOneToManyCompositeKeyEdge()

	parent1ID := 7
	parent2ID := 7
	parents := []any{
		&nullToOneParent{ID: 1, OwnerID: &parent1ID},
		&nullToOneParent{ID: 2, OwnerID: &parent2ID},
	}
	children := []any{
		&nullToOneChild{ID: 1, RefID: 7}, // matches parent 1 [1,7]
		&nullToOneChild{ID: 2, RefID: 7}, // matches parent 2 [2,7]
		&nullToOneChild{ID: 1, RefID: 8}, // [1,8]: partial match with parent 1 [1,7], but a different group
	}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p1 := parents[0].(*nullToOneParent)
	if len(p1.R.Kids) != 1 {
		t.Fatalf("parent1[1,7] should have 1 child [1,7]: got %d children %v", len(p1.R.Kids), p1.R.Kids)
	}
	if p1.R.Kids[0].ID != 1 || p1.R.Kids[0].RefID != 7 {
		t.Fatalf("parent1[1,7] should match child[1,7]: got %#v", p1.R.Kids[0])
	}
	if !p1.R.Loaded.Kids {
		t.Fatalf("parent1 R.Loaded.Kids should be true: got false")
	}

	p2 := parents[1].(*nullToOneParent)
	if len(p2.R.Kids) != 1 {
		t.Fatalf("parent2[2,7] should have 1 child [2,7]: got %d children %v", len(p2.R.Kids), p2.R.Kids)
	}
	if p2.R.Kids[0].ID != 2 || p2.R.Kids[0].RefID != 7 {
		t.Fatalf("parent2[2,7] should match child[2,7]: got %#v", p2.R.Kids[0])
	}
	if !p2.R.Loaded.Kids {
		t.Fatalf("parent2 R.Loaded.Kids should be true: got false")
	}
}
