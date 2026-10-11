package sectionload

// R-24 (P-34-multihop-count.md) §8 T-3/T-4: negative-side tests for
// validateHopChain (validate.go) applied to a multi-hop CountEdge, and a
// positive-side test confirming a single-hop count still passes.
//
// These checks are exercised through All(), never Build() — the same
// reasoning p05_validate_test.go documents for row edges: Build() (run.go)
// deliberately never calls validatePlan, so a broken multi-hop CountEdge
// passed to Build() renders SQL and returns no error. p05NeverCalledExecutor
// (p05_validate_test.go) is reused here: each fixture must be rejected
// BEFORE any query is built or executed.

import (
	"context"
	"errors"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// TestR24Validate_HopColumnLengthMismatchIsRejected (T-3, V-1): Sides[1]
// has 2 ToCols but only 1 FromCols — edgeJoinChain would index past the
// end of the shorter list. validatePlan's n.count branch calls
// validateHopChain(n.count.sides(), ...) directly (validate.go).
func TestR24Validate_HopColumnLengthMismatchIsRejected(t *testing.T) {
	edge := p05TwoHopCountEdge()
	edge.Sides[1].ToCols = []string{"id", "extra"}

	_, err := All(
		context.Background(), p05NeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
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

// TestR24Validate_BrokenTableChainIsRejected (T-3, V-4): Sides[0].ToTable
// no longer matches Sides[1].FromTable.
func TestR24Validate_BrokenTableChainIsRejected(t *testing.T) {
	edge := p05TwoHopCountEdge()
	edge.Sides[1].FromTable = "not_p05_mid"

	_, err := All(
		context.Background(), p05NeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
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

// TestR24Validate_BrokenSchemaChainIsRejected (T-3, V-5): Sides[0].ToSchema
// no longer matches Sides[1].FromSchema.
func TestR24Validate_BrokenSchemaChainIsRejected(t *testing.T) {
	edge := p05TwoHopCountEdge()
	edge.Sides[0].ToSchema = "other_schema"

	_, err := All(
		context.Background(), p05NeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
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

// TestR24Validate_MissingNewCarriedBufIsRejected (self-review Important-1,
// 2026-08-20): a multi-hop CountEdge with NewCarriedBuf left nil — without
// validateMultiHopCountEdge (validate.go), this would sail through
// validatePlan undetected and panic later inside exec.go
// (newCarriedBufAny() called on a nil interface), breaking the "All()
// rejects a broken multi-hop edge before any query runs" guarantee row
// edges already have (p05_validate_test.go's
// TestP05Validate_MissingNewCarriedBufIsRejected).
func TestR24Validate_MissingNewCarriedBufIsRejected(t *testing.T) {
	edge := p05TwoHopCountEdge()
	edge.NewCarriedBuf = nil

	_, err := All(
		context.Background(), p05NeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
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

// TestR24Validate_KeyColsLengthMismatchIsRejected (self-review
// Important-1, 2026-08-20): a multi-hop CountEdge whose KeyCols is shorter
// than Sides[0].FromCols — carriedScanCols() (node.go) would index
// fromCols[i] past the end of KeyCols were this to reach exec.go
// unchecked (the count-edge counterpart of Edge.Carried's length check).
func TestR24Validate_KeyColsLengthMismatchIsRejected(t *testing.T) {
	edge := p05TwoHopCountEdge()
	edge.KeyCols = nil // Sides[0].FromCols has 1 entry ("id"); KeyCols now has 0.

	_, err := All(
		context.Background(), p05NeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
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

// TestR24Validate_SingleHopCountStillPasses (T-4): a single-hop CountEdge
// (len(sides())==1) must take validateHopChain's early-return nil path —
// the pre-R-24 behavior (validatePlan's n.count branch used to always
// return nil unconditionally; it now delegates to validateHopChain, which
// must be a no-op for the common single-hop case). validatePlan is called
// directly (not via All()): a single-hop count's own query would go on to
// execute against p05NeverCalledExecutor and fail the test there instead,
// which would test the wrong thing (this test is only about
// validatePlan's own verdict, not end-to-end execution).
func TestR24Validate_SingleHopCountStillPasses(t *testing.T) {
	edge := p04CountEdge()
	if len(edge.sides()) != 1 {
		t.Fatalf("fixture precondition failed: p04CountEdge() must be single-hop, got %d sides", len(edge.sides()))
	}

	if err := validatePlan([]Node{NewCountNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan = %v, want nil (a single-hop count must pass validateHopChain unchanged)", err)
	}
}
