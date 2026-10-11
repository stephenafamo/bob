package sectionload

// Tests for multi-hop edge validation and error reporting:
//
// - A multi-hop edge with a restricted ColumnPlan (Only/Except) must not be
//   rejected with MissingKeyFromBufError (Field: ChildKeyFromBuf). A
//   multi-hop edge's child key comes from the carried parent key
//   (childKeyFrom() -> ParentKeyFromBuf), and the generated code never sets
//   ChildKeyFromBuf for multi-hop edges, so requiring it would always reject
//   the restricted x multi-hop combination before execution.
//
// - validateMultiHopEdge() must check that each hop's FromCols/ToCols have
//   the same length. edgeJoinChain() and carriedScanCols() index them
//   positionally, so a length mismatch would lead to a panic or invalid SQL.
//
// - The hop-chaining check must compare Schema as well as ToTable;
//   comparing only the table name would accept a JOIN to the wrong
//   intermediate table when same-named tables exist in different schemas.
//
// - The fail-closed branch in stitch.go (childKeyVals == nil ->
//   CarriedKeyMismatchError) needs a negative test (see multihop_stitch_test.go).
//
// - CarriedKeyMismatchError.Error() must output the Carried/FromCols fields
//   it holds.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// --- F-1 -----------------------------------------------------------------

// multiHopRestrictedTwoHopEdge builds a multiHopTwoHopEdge() variant whose
// ColumnPlan restricts the child (multiHopLeaf) to just "label" — the same
// Only-mode restriction shape the public API exposes. keyCols is nil:
// unlike a single-hop edge, a multi-hop edge's own ChildKey/ToCols never
// come from the child table's own columns at all (the whole premise of
// multi-hop edges — the final table has no column holding the parent's key), so there is
// no child-table "join key column" NewColumnPlan needs to force into
// Internal here.
func multiHopRestrictedTwoHopEdge(t *testing.T) Edge[multiHopParent, multiHopLeaf] {
	t.Helper()
	plan, err := NewColumnPlan(multiHopLeafParts().Columns, []string{"label"}, nil, nil)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 1 || plan.Model[0].Name != "label" {
		t.Fatalf("test setup: plan = %+v, want Model=[label]", plan)
	}
	edge := multiHopTwoHopEdge()
	edge.ColumnPlan = plan
	return edge
}

// TestMultiHopValidate_RestrictedMultiHopEdgeAcceptsParentKeyFromBuf is F-1's
// positive case: a restricted (Only) multi-hop edge that DOES set
// ParentKeyFromBuf passes validatePlan.
func TestMultiHopValidate_RestrictedMultiHopEdgeAcceptsParentKeyFromBuf(t *testing.T) {
	edge := multiHopRestrictedTwoHopEdge(t)
	edge.ParentKeyFromBuf = func(b Buf[multiHopParent]) []any {
		return []any{b.BuildFor(nil).ID}
	}

	if err := validatePlan([]Node{NewNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan: %v, want nil (restricted ColumnPlan + ParentKeyFromBuf must pass for a multi-hop edge)", err)
	}
}

// TestMultiHopValidate_RestrictedMultiHopEdgeReachesSQLBuild extends the
// positive case through to L1 (SQL build): the restricted multi-hop edge must not just pass
// validatePlan, it must also render correct SQL through Build.
func TestMultiHopValidate_RestrictedMultiHopEdgeReachesSQLBuild(t *testing.T) {
	edge := multiHopRestrictedTwoHopEdge(t)
	edge.ParentKeyFromBuf = func(b Buf[multiHopParent]) []any {
		return []any{b.BuildFor(nil).ID}
	}

	if err := validatePlan([]Node{NewNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan: %v, want nil", err)
	}

	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(edge),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(built.SQL, `"_sl_t"."label"`) {
		t.Fatalf("restricted SELECT must still fetch label:\n%s", built.SQL)
	}
	// The restricted ColumnPlan excludes "id" from the model, so it must
	// not appear as a plain (non-carried) column of the child section.
	if strings.Contains(built.SQL, `"_sl_t"."id",`) {
		t.Fatalf("restricted SELECT must not fetch the child table's own id column:\n%s", built.SQL)
	}
}

// TestMultiHopValidate_RestrictedMultiHopEdgeWithoutParentKeyFromBufIsRejected
// is F-1's negative case: a restricted multi-hop edge that leaves
// ParentKeyFromBuf nil is rejected — with MissingKeyFromBufError naming
// ParentKeyFromBuf, NOT ChildKeyFromBuf (the pre-fix, always-wrong
// requirement).
func TestMultiHopValidate_RestrictedMultiHopEdgeWithoutParentKeyFromBufIsRejected(t *testing.T) {
	edge := multiHopRestrictedTwoHopEdge(t)
	// ParentKeyFromBuf deliberately left nil (multiHopTwoHopEdge() does not set
	// it either).

	err := validatePlan([]Node{NewNode(edge)}, nil)
	var want *MissingKeyFromBufError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *MissingKeyFromBufError", err)
	}
	if want.Field != "ParentKeyFromBuf" {
		t.Fatalf("Field = %q, want %q (must not require ChildKeyFromBuf for a multi-hop edge)", want.Field, "ParentKeyFromBuf")
	}
	if want.RelName != "Leaves" || want.RestrictedRelName != "Leaves" {
		t.Fatalf("err = %+v, want RelName=RestrictedRelName=Leaves", want)
	}
}

// --- F-2 -------------------------------------------------------------

// TestMultiHopValidate_SideColumnLengthInvariants pins F-2: every Side's own
// FromCols/ToCols must be non-empty and equal in length, or
// edgeJoinChain (compose.go) / carriedScanCols() (node.go) — both of
// which index the two lists positionally with no bounds check — would
// panic or silently build a malformed JOIN condition.
func TestMultiHopValidate_SideColumnLengthInvariants(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(e *Edge[multiHopParent, multiHopLeaf])
	}{
		{
			name: "side0 ToCols longer than FromCols",
			mutate: func(e *Edge[multiHopParent, multiHopLeaf]) {
				e.Sides[0].ToCols = append(e.Sides[0].ToCols, "extra_col")
			},
		},
		{
			name: "side1 FromCols empty",
			mutate: func(e *Edge[multiHopParent, multiHopLeaf]) {
				e.Sides[1].FromCols = nil
			},
		},
		{
			name: "side1 ToCols empty",
			mutate: func(e *Edge[multiHopParent, multiHopLeaf]) {
				e.Sides[1].ToCols = nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			edge := multiHopTwoHopEdge()
			tc.mutate(&edge)

			err := validatePlan([]Node{NewNode(edge)}, nil)
			var want *CarriedKeyMismatchError
			if !errors.As(err, &want) {
				t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
			}
		})
	}
}

// --- F-3 -------------------------------------------------------------

// TestMultiHopValidate_HopSchemaMismatchIsRejected pins F-3: the hop-chain
// check must also compare Schema, not just Table — a same-named table in
// a different schema is a different table, and accepting the mismatch
// would silently JOIN into the wrong physical table.
func TestMultiHopValidate_HopSchemaMismatchIsRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.Sides[0].ToSchema = "schema_a"
	edge.Sides[1].FromSchema = "schema_b"
	// Table names still agree ("mh_mid" on both sides) — only the schema
	// half of the hop's identity differs, pinning that the table-name
	// check alone would have let this through.

	err := validatePlan([]Node{NewNode(edge)}, nil)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	msg := want.Error()
	if !strings.Contains(msg, "schema_a.mh_mid") || !strings.Contains(msg, "schema_b.mh_mid") {
		t.Fatalf("error message must name both schema.table pairs: %q", msg)
	}
}

// TestMultiHopValidate_HopSchemaMatchIsAccepted is the discriminating positive
// case: two hops agreeing on a NON-EMPTY schema must still pass (the
// check compares the two schema strings for equality, it does not
// special-case or reject a non-empty schema outright).
func TestMultiHopValidate_HopSchemaMatchIsAccepted(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.Sides[0].ToSchema = "shared"
	edge.Sides[1].FromSchema = "shared"

	if err := validatePlan([]Node{NewNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan: %v, want nil (both hops agree on schema %q)", err, "shared")
	}
}

// --- F-5 -------------------------------------------------------------

// TestMultiHopStitchInto_MultiHopWithoutChildKeyValsIsRejected is F-5's
// negative test for the stitch.go fail-closed branch that already
// existed (stitchInto: "childKeyVals == nil && len(e.sides()) > 1") but
// had no direct test of its own: calling stitchInto directly with a
// multi-hop edge and a nil childKeyVals must return
// *CarriedKeyMismatchError, never silently fall back to
// e.childKeyAny(c) (which would stitch using the FINAL table's own
// columns instead of the parent's real join key).
func TestMultiHopStitchInto_MultiHopWithoutChildKeyValsIsRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	parents := []any{multiHopParent{ID: 1}}
	children := []any{multiHopLeaf{ID: 10, Label: "a"}}

	_, err := stitchInto(edge, parents, children, nil, nil, nil)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("stitchInto err = %v, want *CarriedKeyMismatchError (a multi-hop edge must reject a nil childKeyVals)", err)
	}
	if want.RelName != "Leaves" {
		t.Fatalf("err = %+v, want RelName=Leaves", want)
	}
}

// --- F-6 -------------------------------------------------------------

// TestMultiHopCarriedKeyMismatchError_MessageIncludesCounts pins F-6:
// CarriedKeyMismatchError.Error() must render the Carried/FromCols
// values it carries, not just the free-text Reason.
func TestMultiHopCarriedKeyMismatchError_MessageIncludesCounts(t *testing.T) {
	err := &CarriedKeyMismatchError{
		RelName:  "Leaves",
		Carried:  2,
		FromCols: 1,
		Reason:   "len(Carried) must equal len(Sides[0].FromCols)",
	}
	msg := err.Error()
	if !strings.Contains(msg, "carried=2") || !strings.Contains(msg, "fromCols=1") {
		t.Fatalf("Error() = %q, want it to include carried=2 and fromCols=1", msg)
	}
	if !strings.Contains(msg, err.Reason) {
		t.Fatalf("Error() = %q, want it to still include the Reason text %q", msg, err.Reason)
	}
}
