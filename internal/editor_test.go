package internal

import (
	"errors"
	"testing"
)

var (
	_ EditRule = deleteRule{}
	_ EditRule = insertRule{}
	_ EditRule = replaceRule{}
)

func TestEditor(t *testing.T) {
	tests := map[string]struct {
		original string
		rules    []EditRule
		expected string
		err      error
	}{
		"delete": {
			original: "hello world",
			rules:    []EditRule{Delete(5, 10)},
			expected: "hello",
		},
		"insert": {
			original: "hello world",
			rules:    []EditRule{Insert(6, "beautiful ")},
			expected: "hello beautiful world",
		},
		"replace": {
			original: "hello world",
			rules:    []EditRule{Replace(6, 10, "beautiful")},
			expected: "hello beautiful",
		},
		"insert-after-delete": {
			original: "hello world",
			rules:    []EditRule{Delete(6, 10), Insert(6, "beautiful")},
			expected: "hello beautiful",
		},
		"insert-before-delete": {
			original: "hello world",
			rules:    []EditRule{Insert(6, "beautiful"), Delete(6, 10)},
			expected: "hello beautiful",
		},
		"out-of-bounds-delete-end": {
			original: "hello world",
			rules:    []EditRule{Delete(6, 11)},
			err:      OutOfBoundsError(11),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := EditString(tc.original, tc.rules...)
			if err != nil {
				if tc.err == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if errors.Is(tc.err, err) {
					t.Fatalf("expected error: %v, got: %v", tc.err, err)
				}
				return
			}

			if actual != tc.expected {
				t.Fatalf("expected: %q, got: %q", tc.expected, actual)
			}
		})
	}
}
