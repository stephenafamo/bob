package gen

import "testing"

func Test_enumValToIdentifier(t *testing.T) {
	tests := []struct {
		val      string
		expected string
	}{
		{"in_progress", "InProgress"},
		{"in-progress", "InProgress"},
		{"in progress", "InProgress"},
		{"IN_PROGRESS", "InProgress"},
		{"in___-__progress", "InProgress"},
		{" in progress ", "InProgress"},
		// This is OK, because enum values are prefixed with the type name, e.g. TaskStatus1InProgress
		{"1-in-progress", "1InProgress"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if actual := enumValToIdentifier(tt.val); actual != tt.expected {
				t.Errorf("enumValToIdentifier(%q) = %q; want %q", tt.val, actual, tt.expected)
			}
		})
	}
}
