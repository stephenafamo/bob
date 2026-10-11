package sectionload

// P-05 Copilot外部レビュー(1回目、issue P-05-multi-hop-relationships.md
// §17)の指摘に対応する修正ラウンドのテスト(F-1〜F-6)。
//
// F-1 (HIGH #1, validate.go:54): 多段edge+restricted ColumnPlan(Only/
// Except)が常にMissingKeyFromBufError(Field: ChildKeyFromBuf)で拒否され
// ていた。多段edgeのchild keyはcarried親キー経由
// (childKeyFrom()→ParentKeyFromBuf)であり、ChildKeyFromBufは生成コード
// (01_edges.go.tpl)が多段では絶対に設定しないフィールドだったため、
// restricted×多段の組み合わせが実行前に必ず弾かれていた。
//
// F-2 (HIGH #2, validate.go:171相当): 各hopのFromCols/ToCols長一致が
// validateMultiHopEdge()で検証されていなかった。edgeJoinChain()や
// carriedScanCols()は位置対応でインデックスするため、長さ不一致は
// panicまたは不正なSQLにつながる。
//
// F-3 (MEDIUM #1, validate.go:186相当): hopの連結検証がToTableのみで
// Schemaを無視していた。同名テーブルが異なるschemaにある場合、誤った
// 中間テーブルへJOINを受理してしまう穴があった。
//
// F-5 (MEDIUM, p05_stitch_test.go): stitch.goのfail-closed分岐
// (childKeyVals == nil → CarriedKeyMismatchError)に対応する否定側テスト
// が無かった。
//
// F-6 (LOW, errors.go:346): CarriedKeyMismatchError.Error()が
// Carried/FromColsフィールドを保持しているのに出力していなかった。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// --- F-1 -----------------------------------------------------------------

// p05RestrictedTwoHopEdge builds a p05TwoHopEdge() variant whose
// ColumnPlan restricts the child (p05Leaf) to just "label" — the same
// Only-mode restriction shape P-03's public API exposes. keyCols is nil:
// unlike a single-hop edge, a multi-hop edge's own ChildKey/ToCols never
// come from the child table's own columns at all (P-05's whole premise —
// the final table has no column holding the parent's key), so there is
// no child-table "join key column" NewColumnPlan needs to force into
// Internal here.
func p05RestrictedTwoHopEdge(t *testing.T) Edge[p05Parent, p05Leaf] {
	t.Helper()
	plan, err := NewColumnPlan(p05LeafParts().Columns, []string{"label"}, nil, nil)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 1 || plan.Model[0].Name != "label" {
		t.Fatalf("test setup: plan = %+v, want Model=[label]", plan)
	}
	edge := p05TwoHopEdge()
	edge.ColumnPlan = plan
	return edge
}

// TestP05Validate_RestrictedMultiHopEdgeAcceptsParentKeyFromBuf is F-1's
// positive case: a restricted (Only) multi-hop edge that DOES set
// ParentKeyFromBuf passes validatePlan.
func TestP05Validate_RestrictedMultiHopEdgeAcceptsParentKeyFromBuf(t *testing.T) {
	edge := p05RestrictedTwoHopEdge(t)
	edge.ParentKeyFromBuf = func(b Buf[p05Parent]) []any {
		return []any{b.BuildFor(nil).ID}
	}

	if err := validatePlan([]Node{NewNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan: %v, want nil (restricted ColumnPlan + ParentKeyFromBuf must pass for a multi-hop edge)", err)
	}
}

// TestP05Validate_RestrictedMultiHopEdgeReachesSQLBuild extends the
// positive case through to L1 (SQL build, issue §17.3 F-1's test
// requirement): the restricted multi-hop edge must not just pass
// validatePlan, it must also render correct SQL through Build.
func TestP05Validate_RestrictedMultiHopEdgeReachesSQLBuild(t *testing.T) {
	edge := p05RestrictedTwoHopEdge(t)
	edge.ParentKeyFromBuf = func(b Buf[p05Parent]) []any {
		return []any{b.BuildFor(nil).ID}
	}

	if err := validatePlan([]Node{NewNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan: %v, want nil", err)
	}

	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
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

// TestP05Validate_RestrictedMultiHopEdgeWithoutParentKeyFromBufIsRejected
// is F-1's negative case: a restricted multi-hop edge that leaves
// ParentKeyFromBuf nil is rejected — with MissingKeyFromBufError naming
// ParentKeyFromBuf, NOT ChildKeyFromBuf (the pre-fix, always-wrong
// requirement).
func TestP05Validate_RestrictedMultiHopEdgeWithoutParentKeyFromBufIsRejected(t *testing.T) {
	edge := p05RestrictedTwoHopEdge(t)
	// ParentKeyFromBuf deliberately left nil (p05TwoHopEdge() does not set
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

// TestP05Validate_SideColumnLengthInvariants pins F-2: every Side's own
// FromCols/ToCols must be non-empty and equal in length, or
// edgeJoinChain (compose.go) / carriedScanCols() (node.go) — both of
// which index the two lists positionally with no bounds check — would
// panic or silently build a malformed JOIN condition.
func TestP05Validate_SideColumnLengthInvariants(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(e *Edge[p05Parent, p05Leaf])
	}{
		{
			name: "side0 ToCols longer than FromCols",
			mutate: func(e *Edge[p05Parent, p05Leaf]) {
				e.Sides[0].ToCols = append(e.Sides[0].ToCols, "extra_col")
			},
		},
		{
			name: "side1 FromCols empty",
			mutate: func(e *Edge[p05Parent, p05Leaf]) {
				e.Sides[1].FromCols = nil
			},
		},
		{
			name: "side1 ToCols empty",
			mutate: func(e *Edge[p05Parent, p05Leaf]) {
				e.Sides[1].ToCols = nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			edge := p05TwoHopEdge()
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

// TestP05Validate_HopSchemaMismatchIsRejected pins F-3: the hop-chain
// check must also compare Schema, not just Table — a same-named table in
// a different schema is a different table, and accepting the mismatch
// would silently JOIN into the wrong physical table.
func TestP05Validate_HopSchemaMismatchIsRejected(t *testing.T) {
	edge := p05TwoHopEdge()
	edge.Sides[0].ToSchema = "schema_a"
	edge.Sides[1].FromSchema = "schema_b"
	// Table names still agree ("p05_mid" on both sides) — only the schema
	// half of the hop's identity differs, pinning that the table-name
	// check alone would have let this through.

	err := validatePlan([]Node{NewNode(edge)}, nil)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	msg := want.Error()
	if !strings.Contains(msg, "schema_a.p05_mid") || !strings.Contains(msg, "schema_b.p05_mid") {
		t.Fatalf("error message must name both schema.table pairs: %q", msg)
	}
}

// TestP05Validate_HopSchemaMatchIsAccepted is the discriminating positive
// case: two hops agreeing on a NON-EMPTY schema must still pass (the
// check compares the two schema strings for equality, it does not
// special-case or reject a non-empty schema outright).
func TestP05Validate_HopSchemaMatchIsAccepted(t *testing.T) {
	edge := p05TwoHopEdge()
	edge.Sides[0].ToSchema = "shared"
	edge.Sides[1].FromSchema = "shared"

	if err := validatePlan([]Node{NewNode(edge)}, nil); err != nil {
		t.Fatalf("validatePlan: %v, want nil (both hops agree on schema %q)", err, "shared")
	}
}

// --- F-5 -------------------------------------------------------------

// TestP05StitchInto_MultiHopWithoutChildKeyValsIsRejected is F-5's
// negative test for the stitch.go fail-closed branch that already
// existed (stitchInto: "childKeyVals == nil && len(e.sides()) > 1") but
// had no direct test of its own: calling stitchInto directly with a
// multi-hop edge and a nil childKeyVals must return
// *CarriedKeyMismatchError, never silently fall back to
// e.childKeyAny(c) (which would stitch using the FINAL table's own
// columns instead of the parent's real join key).
func TestP05StitchInto_MultiHopWithoutChildKeyValsIsRejected(t *testing.T) {
	edge := p05TwoHopEdge()
	parents := []any{p05Parent{ID: 1}}
	children := []any{p05Leaf{ID: 10, Label: "a"}}

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

// TestP05CarriedKeyMismatchError_MessageIncludesCounts pins F-6:
// CarriedKeyMismatchError.Error() must render the Carried/FromCols
// values it carries, not just the free-text Reason.
func TestP05CarriedKeyMismatchError_MessageIncludesCounts(t *testing.T) {
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
