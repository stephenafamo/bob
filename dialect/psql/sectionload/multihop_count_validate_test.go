package sectionload

// Negative-side tests for validateHopChain (validate.go) applied to a multi-hop CountEdge, and a
// positive-side test confirming a single-hop count still passes.
//
// These checks are exercised through All(), never Build() — the same
// reasoning multihop_validate_test.go documents for row edges: Build() (run.go)
// deliberately never calls validatePlan, so a broken multi-hop CountEdge
// passed to Build() renders SQL and returns no error. multiHopNeverCalledExecutor
// (multihop_validate_test.go) is reused here: each fixture must be rejected
// BEFORE any query is built or executed.

import (
	"context"
	"errors"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// TestMultiHopCountValidate_HopColumnLengthMismatchIsRejected: Sides[1]
// has 2 ToCols but only 1 FromCols — edgeJoinChain would index past the
// end of the shorter list. validatePlan's n.count branch calls
// validateHopChain(n.count.sides(), ...) directly (validate.go).
func TestMultiHopCountValidate_HopColumnLengthMismatchIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	edge.Sides[1].ToCols = []string{"id", "extra"}

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewCountNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestMultiHopCountValidate_BrokenTableChainIsRejected: Sides[0].ToTable
// no longer matches Sides[1].FromTable.
func TestMultiHopCountValidate_BrokenTableChainIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	edge.Sides[1].FromTable = "not_mh_mid"

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewCountNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestMultiHopCountValidate_BrokenSchemaChainIsRejected: Sides[0].ToSchema
// no longer matches Sides[1].FromSchema.
func TestMultiHopCountValidate_BrokenSchemaChainIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	edge.Sides[0].ToSchema = "other_schema"

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewCountNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestMultiHopCountValidate_MissingNewCarriedBufIsRejected: a multi-hop CountEdge with NewCarriedBuf left nil — without
// validateMultiHopCountEdge (validate.go), this would sail through
// validatePlan undetected and panic later inside exec.go
// (newCarriedBufAny() called on a nil interface), breaking the "All()
// rejects a broken multi-hop edge before any query runs" guarantee row
// edges already have (multihop_validate_test.go's
// TestMultiHopValidate_MissingNewCarriedBufIsRejected).
func TestMultiHopCountValidate_MissingNewCarriedBufIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	edge.NewCarriedBuf = nil

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewCountNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestMultiHopCountValidate_KeyColsLengthMismatchIsRejected: a multi-hop CountEdge whose KeyCols is shorter
// than Sides[0].FromCols — carriedScanCols() (node.go) would index
// fromCols[i] past the end of KeyCols were this to reach exec.go
// unchecked (the count-edge counterpart of Edge.Carried's length check).
func TestMultiHopCountValidate_KeyColsLengthMismatchIsRejected(t *testing.T) {
	edge := multiHopTwoHopCountEdge()
	edge.KeyCols = nil // Sides[0].FromCols has 1 entry ("id"); KeyCols now has 0.

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewCountNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "LeavesCount" || want.Carried != 0 || want.FromCols != 1 {
		t.Fatalf("err = %+v, want RelName=LeavesCount Carried=0 FromCols=1", want)
	}
}

// TestMultiHopCountValidate_SingleHopCountStillPasses: a single-hop CountEdge
// (len(sides())==1) must take validateHopChain's early-return nil path —
// validatePlan's n.count branch delegates to validateHopChain, which must be
// a no-op for the common single-hop case. validatePlan is called
// directly (not via All()): a single-hop count's own query would go on to
// execute against multiHopNeverCalledExecutor and fail the test there instead,
// which would test the wrong thing (this test is only about
// validatePlan's own verdict, not end-to-end execution).
func TestMultiHopCountValidate_SingleHopCountStillPasses(t *testing.T) {
	edge := countEdge()
	if len(edge.sides()) != 1 {
		t.Fatalf("fixture precondition failed: countEdge() must be single-hop, got %d sides", len(edge.sides()))
	}

	if err := validatePlan([]Node{NewCountNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan = %v, want nil (a single-hop count must pass validateHopChain unchanged)", err)
	}
}
