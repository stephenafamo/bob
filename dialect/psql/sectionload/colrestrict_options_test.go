package sectionload

// Tests for SelectOptions.Only/Except and the package-level sugar functions.
// The colrestrict_ prefix keeps this file from colliding with the existing
// options_test.go.

import (
	"reflect"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// TestSelectOptions_OnlyReturnsCopy: SelectOptions.Only is a
// value-receiver method — calling it again on the RESULT must not affect
// the earlier value, since Node/Edge values (and the SelectOptions they
// carry) may be reused across multiple All()/Build() calls with different
// options each time (the same "copy, never mutate" principle, applied to
// SelectOptions).
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

// TestOnlyExceptPackageFuncsMatchMethodForm: the package-level
// sugar functions Only/Except must build EXACTLY the same value as
// calling SelectOptions{}.Only/.Except directly. reflect.DeepEqual is
// required: SelectOptions holds slice
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

// TestSelectOptions_MergedWithMods covers the basic composition rule:
// chaining Only/Except onto the result of Mods(...) keeps the mods field
// that Mods accumulated and only adds the only/except fields (the two
// groups of fields are independent).
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
