package sectionload

// P-33 §8 テスト2: SelectOptions.Only/Except・パッケージ関数糖衣。
//
// ファイル名注意: 当初 options_test.go という名前で作成しようとしたが、
// 同名の既存ファイル(P-02 の options_test.go、p11CapEdgeWith を定義し
// options_haskids_test.go から参照されている)を上書きしてしまう事故が
// 発生した(発見時に git checkout HEAD で復元済み、実装記録に記載)。
// 衝突を避けるため p33_ prefix を付けたファイル名にした。

import (
	"reflect"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// TestSelectOptions_OnlyReturnsCopy is test 2(a): SelectOptions.Only is a
// value-receiver method — calling it again on the RESULT must not affect
// the earlier value, since Node/Edge values (and the SelectOptions they
// carry) may be reused across multiple All()/Build() calls with different
// options each time (6.4節と同じ「コピーであって破壊しない」原則の
// SelectOptions版)。
func TestSelectOptions_OnlyReturnsCopy(t *testing.T) {
	first := SelectOptions{}.Only("a", "b")
	second := first.Only("c")

	if len(first.only) != 2 || first.only[0] != "a" || first.only[1] != "b" {
		t.Fatalf("first.only = %v, want [a b] (must be unaffected by the later call)", first.only)
	}
	if !first.onlySet {
		t.Fatal("first.onlySet = false, want true")
	}
	if len(second.only) != 1 || second.only[0] != "c" {
		t.Fatalf("second.only = %v, want [c]", second.only)
	}
}

// TestSelectOptions_ExceptReturnsCopy is Except's counterpart to the above.
func TestSelectOptions_ExceptReturnsCopy(t *testing.T) {
	first := SelectOptions{}.Except("x")
	second := first.Except("y", "z")

	if len(first.except) != 1 || first.except[0] != "x" {
		t.Fatalf("first.except = %v, want [x] (must be unaffected by the later call)", first.except)
	}
	if !first.exceptSet {
		t.Fatal("first.exceptSet = false, want true")
	}
	if len(second.except) != 2 || second.except[0] != "y" || second.except[1] != "z" {
		t.Fatalf("second.except = %v, want [y z]", second.except)
	}
}

// TestOnlyExceptPackageFuncsMatchMethodForm is test 2(b): the package-level
// sugar functions Only/Except (6.9節) must build EXACTLY the same value as
// calling SelectOptions{}.Only/.Except directly. reflect.DeepEqual is
// required (I6-1、Opus review 2026-08-18): SelectOptions holds slice
// fields (only/except/mods), which == cannot compare. This is a test
// inside the sectionload package itself, so the unexported fields are
// directly comparable via DeepEqual without any accessor.
func TestOnlyExceptPackageFuncsMatchMethodForm(t *testing.T) {
	gotOnly := Only("a", "b")
	wantOnly := SelectOptions{}.Only("a", "b")
	if !reflect.DeepEqual(gotOnly, wantOnly) {
		t.Fatalf("Only(\"a\",\"b\") = %+v, want %+v", gotOnly, wantOnly)
	}

	gotExcept := Except("x")
	wantExcept := SelectOptions{}.Except("x")
	if !reflect.DeepEqual(gotExcept, wantExcept) {
		t.Fatalf("Except(\"x\") = %+v, want %+v", gotExcept, wantExcept)
	}
}

// TestSelectOptions_MergedWithMods is judgment 1 (6.9節) の合成方針の
// 基本形: Mods(...) の戻り値に Only/Except をチェーンしても、Mods が
// 積んだ mods フィールドは保持されたまま、only/except 系フィールドだけが
// 追加される(両フィールド群が独立していることの確認)。
func TestSelectOptions_MergedWithMods(t *testing.T) {
	m := sm.Limit(5)
	base := Mods(m)
	merged := base.Except("secret")

	if len(merged.mods) != 1 {
		t.Fatalf("merged.mods = %v, want the 1 mod Mods(...) produced (Except must not clear it)", merged.mods)
	}
	if !merged.exceptSet || len(merged.except) != 1 || merged.except[0] != "secret" {
		t.Fatalf("merged = %+v, want exceptSet=true except=[secret]", merged)
	}
}
