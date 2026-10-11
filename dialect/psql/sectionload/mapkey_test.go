package sectionload

// mapKey must not rewrite the []byte elements of its vals argument.
// Mutating a throwaway slice would be harmless, but hooks.go's runThen
// appends the kv it passed to mapKey straight into keys, and those keys
// become the match values (SQL parameters) of the Then boundary's
// buildEdgeSelect. The kv passed from stitch.go is likewise an element of a
// persistent slice (res.parentKeys) that execIsland accumulated via
// cloneKey, so the argument is not disposable and a mutation would leak
// into SQL parameters.
//
// This file pins the "mapKey does not mutate its argument" contract without
// a database, as the counterpart of clonekey_test.go, which pins cloneKey's
// contract (deep-copying []byte when a key is stored).
//
// Important: the premise of these tests is that the elements passed in really
// are []byte. If the premise breaks (a non-[]byte is passed), a mutating
// implementation would pass unnoticed, so mapKeyRequireByteElem checks the
// premise before every call.

import "testing"

// mapKeyRequireByteElem checks the test premise; call it before mapKey.
func mapKeyRequireByteElem(t *testing.T, vals []any, i int) {
	t.Helper()
	if _, ok := vals[i].([]byte); !ok {
		t.Fatalf("test premise broken: vals[%d] must be []byte but is %T", i, vals[i])
	}
}

// mapKeyAssertByteElem checks that the element is still a []byte with the
// same value after mapKey was called. A mutating implementation fails here.
func mapKeyAssertByteElem(t *testing.T, vals []any, i int, want string) {
	t.Helper()
	b, ok := vals[i].([]byte)
	if !ok {
		t.Fatalf("mapKey must not modify its argument: vals[%d] changed type from []byte to %T (value: %v)", i, vals[i], vals[i])
	}
	if string(b) != want {
		t.Fatalf("mapKey must not modify its argument: vals[%d] changed value from %q to %q", i, want, string(b))
	}
}

// Single key: the argument is unchanged even when it holds a []byte element.
func TestMapKey_DoesNotMutateSingleByteKeyArgument(t *testing.T) {
	vals := []any{[]byte("ab")}
	mapKeyRequireByteElem(t, vals, 0)

	k, err := mapKey(vals)
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	if k != "ab" {
		t.Fatalf("mapKey() = %#v, want %q", k, "ab")
	}
	mapKeyAssertByteElem(t, vals, 0, "ab")
}

// Composite key: the []byte elements are placed after the first position
// (index 1 and 2), because checking only index 0 would let an implementation
// that protects just the first element pass. It also checks that the
// non-[]byte element (the int at index 0) is untouched.
func TestMapKey_DoesNotMutateCompositeKeyArgument(t *testing.T) {
	vals := []any{1, []byte("ab"), []byte("cd")}
	mapKeyRequireByteElem(t, vals, 1)
	mapKeyRequireByteElem(t, vals, 2)

	k, err := mapKey(vals)
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	if want := ([3]any{1, "ab", "cd"}); k != want {
		t.Fatalf("mapKey() = %#v, want %#v", k, want)
	}
	if _, ok := vals[0].(int); !ok {
		t.Fatalf("mapKey must not modify non-[]byte elements either: vals[0] changed type from int to %T", vals[0])
	}
	mapKeyAssertByteElem(t, vals, 1, "ab")
	mapKeyAssertByteElem(t, vals, 2, "cd")
}

// The return value keeps its documented shape (packed form for widths 1 to
// 4). This pins unchanged behavior.
func TestMapKey_ReturnsComparableKeyForEveryWidth(t *testing.T) {
	cases := []struct {
		name string
		vals []any
		want any
	}{
		{"width1", []any{[]byte("ab")}, "ab"},
		{"width2", []any{1, []byte("ab")}, [2]any{1, "ab"}},
		{"width3", []any{1, "x", []byte("ab")}, [3]any{1, "x", "ab"}},
		{"width4", []any{1, "x", []byte("ab"), 2}, [4]any{1, "x", "ab", 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, err := mapKey(tc.vals)
			if err != nil {
				t.Fatalf("mapKey(%#v) error = %v, want nil", tc.vals, err)
			}
			if k != tc.want {
				t.Fatalf("mapKey(%#v) = %#v, want %#v", tc.vals, k, tc.want)
			}
		})
	}
}

// []byte values with the same content (backed by different arrays) and an
// equal string must land in the same map entry. This is the reason mapKey
// exists and the most important property to preserve.
func TestMapKey_EqualByteKeysMapToTheSameEntry(t *testing.T) {
	k1, err := mapKey([]any{[]byte("ab"), 7})
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	k2, err := mapKey([]any{append([]byte(nil), 'a', 'b'), 7})
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}
	k3, err := mapKey([]any{"ab", 7})
	if err != nil {
		t.Fatalf("mapKey() error = %v, want nil", err)
	}

	m := map[any]int{}
	m[k1]++
	m[k2]++
	m[k3]++
	if len(m) != 1 || m[k1] != 3 {
		t.Fatalf("equal keys must collapse into one entry: map = %#v (k1=%#v k2=%#v k3=%#v)", m, k1, k2, k3)
	}
}

// Key width over the limit: the error message and return value are as
// documented, and the argument is not modified before the error is returned
// (an implementation that converts elements in a loop before checking the
// width would mutate it).
func TestMapKey_UnsupportedWidthErrsWithoutTouchingArgument(t *testing.T) {
	contents := []string{"a", "b", "c", "d", "e"}
	vals := make([]any, len(contents))
	for i, c := range contents {
		vals[i] = []byte(c)
		mapKeyRequireByteElem(t, vals, i)
	}

	k, err := mapKey(vals)
	if k != nil {
		t.Fatalf("mapKey() key = %#v, want nil on error", k)
	}
	if err == nil {
		t.Fatal("mapKey() error = nil, want an unsupported-width error")
	}
	const want = "unsupported key width 5 (max 4)"
	if err.Error() != want {
		t.Fatalf("mapKey() error = %q, want %q", err, want)
	}
	for i, c := range contents {
		mapKeyAssertByteElem(t, vals, i, c)
	}
}

// A key width of 0 (including nil) is also an error.
func TestMapKey_ZeroWidthErrs(t *testing.T) {
	k, err := mapKey(nil)
	if k != nil {
		t.Fatalf("mapKey(nil) key = %#v, want nil on error", k)
	}
	if err == nil {
		t.Fatal("mapKey(nil) error = nil, want an unsupported-width error")
	}
	const want = "unsupported key width 0 (max 4)"
	if err.Error() != want {
		t.Fatalf("mapKey(nil) error = %q, want %q", err, want)
	}
}
