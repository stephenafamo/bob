package sectionload

// Confirmation tests (no code change involved) that two existing checks
// already cover the multi-hop count case correctly, because both are
// structural checks that never look at hop count:
//
//   - resolveNode (columnselect.go) checks n.count != nil BEFORE it ever
//     looks at Sides, so "multi-hop count + Only/Except" is
//     CountColumnRestrictionUnsupportedError, never
//     MultiHopColumnRestrictionUnsupportedError — CountEdge has no
//     ColumnPlan field at all, multi-hop or not.
//   - validatePlan's PreloadCount/count-Node conflict check (validate.go)
//     matches purely by RelName string, never by hop count, so a TOP-LEVEL
//     multi-hop count Node colliding with a same-named PreloadCount
//     declaration is caught exactly like a single-hop one.

import (
	"errors"
	"testing"
)

// TestMultiHopCountColumnSelect_MultiHopCountWithOnlyIsRejected: a
// multi-hop CountEdge with Only(...) set must be
// CountColumnRestrictionUnsupportedError — never
// MultiHopColumnRestrictionUnsupportedError, since resolveNode
// (columnselect.go) branches on n.count != nil before it ever inspects
// Sides.
func TestMultiHopCountColumnSelect_MultiHopCountWithOnlyIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	edge.Options = Only("id")

	_, err := resolveColumnPlans([]Node{NewCountNode(edge)})
	var want *CountColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountColumnRestrictionUnsupportedError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// declaredCountPreloadFake stands in for a bob PreloadCount declaration
// read off a caller's parentQuery (declaredCountPreloads,
// countpreload.go) — validatePlan (validate.go) only ever reads RelName
// off this struct for its conflict check, so a literal is enough here;
// the SQL-parsing half (declaredCountPreloadsShim) is covered separately
// by countpreload_test.go and is not what this test exercises.
func declaredCountPreloadFake(relName string) declaredCountPreload {
	return declaredCountPreload{RelName: relName, Alias: "__count_" + relName}
}

// TestMultiHopCountValidate_TopLevelMultiHopCountConflictsWithPreloadCount: a TOP-LEVEL multi-hop count Node whose RelName collides with a
// declared PreloadCount must be rejected, exactly like the existing
// single-hop case — validatePlan's conflict check (validate.go) matches
// purely on n.count.relName() vs declaredNames, never on hop count.
func TestMultiHopCountValidate_TopLevelMultiHopCountConflictsWithPreloadCount(t *testing.T) {
	edge := multiHopTwoHopCountEdge() // RelName: "LeavesCount"

	err := validatePlan(
		[]Node{NewCountNode(edge)},
		[]declaredCountPreload{declaredCountPreloadFake("LeavesCount")},
	)
	var want *PreloadCountConflictError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *PreloadCountConflictError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestMultiHopCountValidate_TopLevelMultiHopCountNoConflictWithDifferentName is the
// negative control for the test above: a different RelName must not
// conflict, confirming the match really is by name (not merely "any
// multi-hop count Node vs any declared PreloadCount").
func TestMultiHopCountValidate_TopLevelMultiHopCountNoConflictWithDifferentName(t *testing.T) {
	edge := multiHopTwoHopCountEdge() // RelName: "LeavesCount"

	err := validatePlan(
		[]Node{NewCountNode(edge)},
		[]declaredCountPreload{declaredCountPreloadFake("SomethingElseCount")},
	)
	if err != nil {
		t.Fatalf("err = %v, want nil (different RelName must not conflict)", err)
	}
}
