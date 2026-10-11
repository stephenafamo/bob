package sectionload

import (
	"errors"
	"testing"
)

// NewColumnPlan: 明示選択されていない結合キーを内部列として保持する契約。

func TestNewColumnPlan_NoRestrictionReturnsZeroValue(t *testing.T) {
	plan, err := NewColumnPlan([]Column{{Name: "id"}, {Name: "name"}}, nil, nil, []string{"id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 0 || len(plan.Internal) != 0 {
		t.Fatalf("plan = %+v, want the zero value (no restriction)", plan)
	}
	if all := plan.All(); all != nil {
		t.Fatalf("All() = %v, want nil", all)
	}
}

func TestNewColumnPlan_OnlyAutoIncludesUnmentionedKey(t *testing.T) {
	all := []Column{{Name: "id"}, {Name: "parent_id"}, {Name: "secret"}}
	plan, err := NewColumnPlan(all, []string{"id"}, nil, []string{"parent_id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 1 || plan.Model[0].Name != "id" {
		t.Fatalf("Model = %v, want [id]", plan.Model)
	}
	if len(plan.Internal) != 1 || plan.Internal[0].Name != "parent_id" {
		t.Fatalf("Internal = %v, want [parent_id] (key auto-included, not requested)", plan.Internal)
	}
}

func TestNewColumnPlan_OnlyExplicitlyIncludingKeyStaysInModel(t *testing.T) {
	all := []Column{{Name: "id"}, {Name: "parent_id"}, {Name: "secret"}}
	plan, err := NewColumnPlan(all, []string{"id", "parent_id"}, nil, []string{"parent_id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Internal) != 0 {
		t.Fatalf("Internal = %v, want empty: the caller asked for the key explicitly, it belongs in Model only", plan.Internal)
	}
	if len(plan.Model) != 2 || plan.Model[0].Name != "id" || plan.Model[1].Name != "parent_id" {
		t.Fatalf("Model = %v, want [id parent_id] in table order", plan.Model)
	}
}

// TestNewColumnPlan_ExceptUnmentionedKeyStaysInModel is P-33's regression
// test for the except-mode semantics change (parts.go doc, "P-33"): a key
// column not named in except now lands in Model, matching bob's own
// psql.PreloadExcept. This REPLACES the pre-P-33 contract this test used
// to assert (the key always routed to Internal in except mode) — see
// P-33-relation-column-restriction.md §6.5 for why the contract changed.
func TestNewColumnPlan_ExceptUnmentionedKeyStaysInModel(t *testing.T) {
	all := []Column{{Name: "id"}, {Name: "parent_id"}, {Name: "secret"}}
	plan, err := NewColumnPlan(all, nil, []string{"secret"}, []string{"parent_id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 2 || plan.Model[0].Name != "id" || plan.Model[1].Name != "parent_id" {
		t.Fatalf("Model = %v, want [id, parent_id] (secret excluded; parent_id is a key not named in except, so it stays in Model)", plan.Model)
	}
	if len(plan.Internal) != 0 {
		t.Fatalf("Internal = %v, want empty: parent_id is not excluded, so it never routes to Internal", plan.Internal)
	}
}

func TestNewColumnPlan_ExceptExcludingKeyIsError(t *testing.T) {
	all := []Column{{Name: "id"}, {Name: "parent_id"}, {Name: "secret"}}
	_, err := NewColumnPlan(all, nil, []string{"parent_id"}, []string{"parent_id"})
	var keyErr *KeyColumnExcludedError
	if !errors.As(err, &keyErr) || keyErr.Column != "parent_id" {
		t.Fatalf("err = %v, want *KeyColumnExcludedError{Column: \"parent_id\"}", err)
	}
}

func TestNewColumnPlan_OnlyAndExceptAreMutuallyExclusive(t *testing.T) {
	all := []Column{{Name: "id"}}
	_, err := NewColumnPlan(all, []string{"id"}, []string{"id"}, nil)
	var conflictErr *ColumnSelectionConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("err = %v, want *ColumnSelectionConflictError", err)
	}
}

func TestNewColumnPlan_UnknownKeyColumnIsError(t *testing.T) {
	all := []Column{{Name: "id"}}
	_, err := NewColumnPlan(all, []string{"id"}, nil, []string{"does_not_exist"})
	var keyErr *UnknownKeyColumnError
	if !errors.As(err, &keyErr) || keyErr.Column != "does_not_exist" {
		t.Fatalf("err = %v, want *UnknownKeyColumnError{Column: \"does_not_exist\"}", err)
	}
}

// TestNewColumnPlan_UnknownKeyColumnAlsoNamedInOnlyIsStillAnError is the
// regression test for issue P-03's Important finding (Codex review,
// 2026-08-09): when the same unknown name appears in both Only and
// keyCols, the onlyMode/selected shortcut used to skip byName validation
// entirely before this fix — returning a plan with neither Model nor
// Internal populated instead of the UnknownKeyColumnError a stale
// generated-code reference should produce.
func TestNewColumnPlan_UnknownKeyColumnAlsoNamedInOnlyIsStillAnError(t *testing.T) {
	all := []Column{{Name: "id"}}
	_, err := NewColumnPlan(all, []string{"missing"}, nil, []string{"missing"})
	var keyErr *UnknownKeyColumnError
	if !errors.As(err, &keyErr) || keyErr.Column != "missing" {
		t.Fatalf("err = %v, want *UnknownKeyColumnError{Column: \"missing\"}", err)
	}
}

func TestNewColumnPlan_UnknownKeyColumnWithoutRestrictionIsError(t *testing.T) {
	all := []Column{{Name: "id", DBType: "int4"}}
	_, err := NewColumnPlan(all, nil, nil, []string{"missing"})
	var want *UnknownKeyColumnError
	if !errors.As(err, &want) || want.Column != "missing" {
		t.Fatalf("expected UnknownKeyColumnError{missing}, got %v", err)
	}
}

func TestNewColumnPlan_UnknownKeyColumnNamedInExceptIsUnknownBeforeExcluded(t *testing.T) {
	all := []Column{{Name: "id", DBType: "int4"}}
	_, err := NewColumnPlan(all, nil, []string{"missing"}, []string{"missing"})
	var want *UnknownKeyColumnError
	if !errors.As(err, &want) || want.Column != "missing" {
		t.Fatalf("stale keyCols must be identified as unknown, not merely excluded; got %v", err)
	}
}

// TestNewColumnPlan_ExceptAllColumnsKeepsEmptyModelDistinctFromUnrestricted
// is P-03 round 2's Important 3 regression test, re-scoped for P-33: after
// the except-mode semantics change (§6.5), a key column can no longer be
// used to force an empty Model (an unexcluded key always lands in Model
// now), so this uses a table with NO key columns and excludes everything
// instead — the only way except mode can still select zero Model columns.
func TestNewColumnPlan_ExceptAllColumnsKeepsEmptyModelDistinctFromUnrestricted(t *testing.T) {
	all := []Column{{Name: "a", DBType: "int4"}, {Name: "b", DBType: "int4"}}
	plan, err := NewColumnPlan(all, nil, []string{"a", "b"}, nil)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if plan.Model == nil {
		t.Fatal("restricted plan must have a NON-nil (possibly empty) Model: nil means unrestricted")
	}
	if len(plan.Model) != 0 {
		t.Fatalf("Model should be empty here, got %+v", plan.Model)
	}
	if len(plan.Internal) != 0 {
		t.Fatalf("Internal should be empty here (no key columns), got %+v", plan.Internal)
	}
}

// --- Buf-level mechanism (P-03 §11.3項目9・10): an Internal column can be
// fetched into the scan buffer, read for stitching via *KeyFromBuf, and
// never copied into the model — even when the model has no field for it
// at all, unlike bob's own multi-side side-channel design.

// fakeChild deliberately has no field for the parent key: proving Internal
// columns need not exist on the model, not merely be left unpopulated.
type fakeChild struct {
	ID     int
	Secret string
}

type fakeChildBuf struct {
	id       int
	secret   string
	parentID int // Internal only
}

func (b *fakeChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.secret, &b.parentID}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "secret":
			out[i] = &b.secret
		case "parent_id":
			out[i] = &b.parentID
		default:
			panic("fakeChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *fakeChildBuf) BuildFor(modelCols []Column) *fakeChild {
	if modelCols == nil {
		return &fakeChild{ID: b.id, Secret: b.secret}
	}
	m := &fakeChild{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "secret":
			m.Secret = b.secret
			// no "parent_id" case: Internal columns are not assignable
			// into fakeChild even if a caller mistakenly listed one.
		}
	}
	return m
}

// panicIfFullBuildBuf panics on BuildFor(nil) — the pre-fix sentinel
// collision would call exactly that with an Internal-only plan (P-03 round
// 2 Important 3).
type panicIfFullBuildBuf struct{ fakeChildBuf }

func (b *panicIfFullBuildBuf) BuildFor(modelCols []Column) *fakeChild {
	if modelCols == nil {
		panic("BuildFor(nil) must not be called for an Internal-only plan")
	}
	return &fakeChild{}
}

// TestColumnPlan_InternalOnlyDoesNotCallFullBuild is P-03 round 2's
// Important 3 regression test. Re-scoped to only mode for P-33 (Opus
// review I1, 2026-08-18): after the except-mode semantics change (§6.5),
// except(nil, ["id"], keyCols=["parent_id"]) no longer produces an
// Internal-only plan (parent_id, not named in except, now lands in
// Model) — this test's whole premise would silently stop being
// exercised if left as except mode. only mode still produces the same
// Internal-only shape this test needs to check.
func TestColumnPlan_InternalOnlyDoesNotCallFullBuild(t *testing.T) {
	all := []Column{{Name: "id", DBType: "int4"}, {Name: "parent_id", DBType: "int4"}}
	plan, err := NewColumnPlan(all, []string{"id"}, nil, []string{"parent_id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 1 || plan.Model[0].Name != "id" {
		t.Fatalf("Model = %v, want [id]", plan.Model)
	}
	if len(plan.Internal) != 1 || plan.Internal[0].Name != "parent_id" {
		t.Fatalf("Internal = %v, want [parent_id] (unmentioned key in only mode)", plan.Internal)
	}
	b := &panicIfFullBuildBuf{}
	got := b.BuildFor(plan.Model) // 空ではない([id]) → panicIfFullBuildBuf.BuildFor(nil)には到達しない
	if got == nil {
		t.Fatal("BuildFor(plan.Model) returned nil")
	}
}

// TestColumnPlan_EmptyModelDoesNotCallFullBuild is the direct successor
// to the pre-P-33 except-mode regression test for P-03 round 2 Important
// 3(セルフレビュー Important 1、2026-08-19): 上の
// TestColumnPlan_InternalOnlyDoesNotCallFullBuild は onlyMode へ差し替えた
// 結果 plan.Model が [id](長さ1、空ではない)になってしまい、「制限されて
// いるが公開列が0件(plan.Model が非nilだが長さ0)」というP-03 round2
// Important3本来のシナリオを、P-33後は一切踏まなくなっていた
// (panicIfFullBuildBuf を使うテストはこのパッケージ内でこの2つの関数
// だけであり、他に空Modelケースを検証するテストが無い)。
//
// P-33後の except モードでは、結合キーが1つ以上あるテーブルでは「除外
// されなかったキー列が必ずModelへ残る」ため、Modelを完全に空にすることは
// もうできない(§6.5参照)。したがって、この統合的な検証を復活させる唯一
// の方法は、TestNewColumnPlan_ExceptAllColumnsKeepsEmptyModelDistinct
// FromUnrestricted と同じ「結合キーが無いテーブル」フィクスチャを使い、
// except で全列を除外することである。
func TestColumnPlan_EmptyModelDoesNotCallFullBuild(t *testing.T) {
	all := []Column{{Name: "a", DBType: "int4"}, {Name: "b", DBType: "int4"}}
	plan, err := NewColumnPlan(all, nil, []string{"a", "b"}, nil)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if plan.Model == nil || len(plan.Model) != 0 {
		t.Fatalf("plan.Model = %v, want a non-nil empty slice", plan.Model)
	}
	b := &panicIfFullBuildBuf{}
	got := b.BuildFor(plan.Model) // 非nilの空スライス → BuildFor(nil)には到達しない
	if got == nil {
		t.Fatal("BuildFor(plan.Model) returned nil")
	}
}

type fakeParent struct{ ID int }

// TestColumnPlan_InternalColumnScannedButNeverBuilt is P-03's regression
// test for "an Internal column is scanned into the buffer but never
// copied into the model, even when the model has no field for it at
// all". Re-scoped to only mode for P-33 (Opus review I1, 2026-08-18):
// except(nil, ["secret"], keyCols=["parent_id"]) no longer routes
// parent_id to Internal (it lands in Model now, §6.5), so this test's
// premise would silently stop being exercised if left as except mode.
func TestColumnPlan_InternalColumnScannedButNeverBuilt(t *testing.T) {
	all := []Column{{Name: "id", DBType: "int4"}, {Name: "secret", DBType: "text"}, {Name: "parent_id", DBType: "int4"}}
	plan, err := NewColumnPlan(all, []string{"id", "secret"}, nil, []string{"parent_id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Internal) != 1 || plan.Internal[0].Name != "parent_id" {
		t.Fatalf("Internal = %v, want [parent_id] (unmentioned key in only mode)", plan.Internal)
	}
	buf := &fakeChildBuf{id: 1, secret: "x", parentID: 42}
	targets := buf.TargetsFor(plan.All()) // parent_id もスキャン対象に含まれる
	if len(targets) != 3 {
		t.Fatalf("TargetsFor(plan.All()) len = %d, want 3", len(targets))
	}
	m := buf.BuildFor(plan.Model) // parent_id は含まれない
	if m.ID != 1 || m.Secret != "x" {
		t.Fatalf("m = %+v, want ID=1 Secret=x", m)
	}
	// fakeChild has no ParentID field at all — parent_id being Internal
	// means there is nothing to assert was "left zero"; its absence from
	// the model type itself is the proof it was never copied.
}

func TestColumnPlan_ChildKeyFromBufReadsInternalColumnNoModelFieldExists(t *testing.T) {
	edge := Edge[*fakeParent, *fakeChild]{
		ChildKeyFromBuf: func(b Buf[*fakeChild]) []any {
			return []any{b.(*fakeChildBuf).parentID}
		},
	}
	buf := &fakeChildBuf{id: 7, secret: "s3cr3t", parentID: 42}

	got := edge.childKeyFromBufAny(bufAdapter[*fakeChild]{buf})
	if len(got) != 1 || got[0].(int) != 42 {
		t.Fatalf("childKeyFromBufAny = %v, want [42]", got)
	}
}

func TestColumnPlan_ChildKeyFromBufFallsBackToChildKeyOfFullBuild(t *testing.T) {
	edge := Edge[*fakeParent, *fakeChild]{
		ChildKey: func(c *fakeChild) []any { return []any{c.ID} },
		// ChildKeyFromBuf intentionally nil: exercise the fallback path.
	}
	buf := &fakeChildBuf{id: 7, secret: "s3cr3t", parentID: 42}

	got := edge.childKeyFromBufAny(bufAdapter[*fakeChild]{buf})
	if len(got) != 1 || got[0].(int) != 7 {
		t.Fatalf("childKeyFromBufAny fallback = %v, want [7] (from BuildFor(nil).ID)", got)
	}
}
