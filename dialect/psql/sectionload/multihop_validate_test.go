package sectionload

// Negative-side tests for validateMultiHopEdge
// (validate.go) and the reservedNameRe addition (parentexpr.go).
//
// Every fixture here starts from multiHopTwoHopEdge() (multihop_sql_test.go), whose
// ColumnPlan is the ZERO value — deliberately: a check nested inside validatePlan's
// columnPlanRestricts() branch would only ever fire for a RESTRICTED
// edge, and a multi-hop edge with an unrestricted ColumnPlan (the common
// case — most through relationships select every column) would then
// sail past a broken Carried/NewCarriedBuf/Sides contract undetected.
// These fixtures pin that the check fires regardless.
//
// The Carried/NewCarriedBuf/Sides checks are tested through All(), never
// Build(): Build() (run.go) deliberately does not call validatePlan (its
// own doc comment describes it as "show me the SQL
// sectionload would run", with no pre-flight checks at all), so a broken
// multi-hop Edge passed to Build() renders SQL and returns no error; the
// mismatch would only surface later, as a LayoutError or an out-of-range
// panic, deep inside execution — which Build() never reaches. This
// asymmetry is intentional.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/scan"
)

// multiHopNeverCalledExecutor is used by every All() call in this file: each
// fixture is broken in a way validatePlan (run.go step [1]) must reject
// BEFORE any query is built or executed, so QueryContext/ExecContext
// must never actually be invoked. Calling either is itself a test
// failure — the check under test would then be running too late.
type multiHopNeverCalledExecutor struct{ t *testing.T }

func (e multiHopNeverCalledExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	e.t.Helper()
	e.t.Fatalf("QueryContext must not be called: validatePlan should have rejected this tree first (query: %s)", query)
	return nil, fmt.Errorf("unreachable")
}

func (e multiHopNeverCalledExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	e.t.Helper()
	e.t.Fatalf("ExecContext must not be called: validatePlan should have rejected this tree first (query: %s)", query)
	return nil, fmt.Errorf("unreachable")
}

var _ bob.Executor = multiHopNeverCalledExecutor{}

// TestMultiHopValidate_CarriedLengthMismatchIsRejected: Carried has 2 entries
// but Sides[0].FromCols has 1 — edgeJoinChain/carriedScanCols would index
// past the end of whichever list is shorter.
func TestMultiHopValidate_CarriedLengthMismatchIsRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.Carried = []Column{
		{Name: "_sl_k0", DBType: "int4"},
		{Name: "_sl_k1", DBType: "int4"},
	}

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "Leaves" || want.Carried != 2 || want.FromCols != 1 {
		t.Fatalf("err = %+v, want RelName=Leaves Carried=2 FromCols=1", want)
	}
}

// TestMultiHopValidate_MissingNewCarriedBufIsRejected: a multi-hop edge with
// NewCarriedBuf left nil — execIsland would have no Buf[P] to scan the
// carried column(s) into, and childKeyFrom's multi-hop branch would panic
// on a nil interface type assertion.
func TestMultiHopValidate_MissingNewCarriedBufIsRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.NewCarriedBuf = nil

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "Leaves" {
		t.Fatalf("err = %+v, want RelName=Leaves", want)
	}
}

// TestMultiHopValidate_BrokenSidesChainIsRejected: Sides[0].ToTable
// ("mh_mid") no longer matches Sides[1].FromTable — the hop chain bob's
// own Relationship.Validate() would normally guarantee is broken, so
// edgeJoinChain's alias scheme (which assumes it) cannot be trusted.
func TestMultiHopValidate_BrokenSidesChainIsRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	edge.Sides[1].FromTable = "not_mh_mid"

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError", err)
	}
	if want.RelName != "Leaves" {
		t.Fatalf("err = %+v, want RelName=Leaves", want)
	}
}

// TestMultiHopValidate_UnrestrictedMultiHopEdgeStillValidated is the direct
// discriminator: multiHopTwoHopEdge() carries the ZERO ColumnPlan (unrestricted
// — columnPlanRestricts() is false), and a broken Carried is still
// rejected. If validateMultiHopEdge's call were nested inside
// validatePlan's `if e.columnPlanRestricts()` branch instead of standing
// beside it, this exact case would slip through undetected.
func TestMultiHopValidate_UnrestrictedMultiHopEdgeStillValidated(t *testing.T) {
	edge := multiHopTwoHopEdge()
	if edge.ColumnPlan.Model != nil || edge.ColumnPlan.Internal != nil {
		t.Fatalf("fixture precondition failed: multiHopTwoHopEdge() must have a ZERO ColumnPlan for this test to discriminate anything, got %+v", edge.ColumnPlan)
	}
	edge.NewCarriedBuf = nil

	_, err := All(
		context.Background(), multiHopNeverCalledExecutor{t: t},
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(edge),
	)
	var want *CarriedKeyMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CarriedKeyMismatchError (an unrestricted multi-hop edge must still be validated)", err)
	}
}

// TestMultiHopValidate_ReservedCarriedNameInParentQueryIsRejected pins the
// reservedNameRe addition (parentexpr.go): a parent query that happens to
// select a column literally named "_sl_k0" must be rejected, the same way
// "_sl_set"/"_sl_cnt1"/etc. already were. Build() (not All()) is enough
// here — the reserved-name check runs inside composeIsland's
// parentSource wrapper, reached by bob.Build(ctx, q) at Build's own last
// step, with no executor needed at all.
func TestMultiHopValidate_ReservedCarriedNameInParentQueryIsRejected(t *testing.T) {
	_, err := Build(
		context.Background(),
		psql.RawQuery(`SELECT id, id AS "_sl_k0" FROM mh_parents`),
		multiHopParentParts(),
		NewNode(multiHopTwoHopEdge()),
	)
	var want *ReservedNameError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ReservedNameError", err)
	}
	if want.Name != "_sl_k0" {
		t.Fatalf("ReservedNameError.Name = %q, want %q", want.Name, "_sl_k0")
	}
}
