package sectionload

import "testing"

// TestCloneKey_* pins cloneKey's contract: nothing in Edge's contract requires a
// *KeyFromBuf/ParentKey/ChildKey callback to return a freshly allocated
// slice, and scan buffers are reused row to row, so a non-compliant
// callback returning a slice that aliases reused storage would silently
// corrupt an already-stored key on a later row. cloneKey (exec.go) is the
// single point every key tuple passes through on its way into
// parentKeys/childKeys/thenKeys/rootChildKeys; these tests pin its
// contract directly, independent of execIsland's full row-scan machinery.

func TestCloneKey_NilPassesThrough(t *testing.T) {
	if got := cloneKey(nil); got != nil {
		t.Fatalf("cloneKey(nil) = %v, want nil", got)
	}
}

func TestCloneKey_MutatingSourceAfterwardDoesNotAffectClone(t *testing.T) {
	src := []any{1, "a"}
	got := cloneKey(src)
	src[0] = 999
	src[1] = "z"
	if got[0] != 1 || got[1] != "a" {
		t.Fatalf("cloneKey result = %v, want [1 a] independent of the later-mutated source (this is exactly what protects a stored key from a reused scan buffer)", got)
	}
}

func TestCloneKey_MutatingByteElementAfterwardDoesNotAffectClone(t *testing.T) {
	b := []byte("ab")
	got := cloneKey([]any{b})
	b[0] = 'z'
	if s := string(got[0].([]byte)); s != "ab" {
		t.Fatalf("cloneKey must deep-copy []byte elements (mapKey supports them); got %q", s)
	}
}
