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
		{"start < end", "StartU3CEnd"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if actual := enumValToIdentifier(tt.val); actual != tt.expected {
				t.Errorf("enumValToIdentifier(%q) = %q; want %q", tt.val, actual, tt.expected)
			}
		})
	}
}

func Test_enumValToScreamingSnakeCase(t *testing.T) {
	tests := []struct {
		val      string
		expected string
	}{
		{"in_progress", "IN_PROGRESS"},
		{"in-progress", "IN_PROGRESS"},
		{"in progress", "IN_PROGRESS"},
		{"IN_PROGRESS", "IN_PROGRESS"},
		{"in___-__progress", "IN______PROGRESS"},
		{" in progress ", "_IN_PROGRESS_"},
		{"1-in-progress", "1_IN_PROGRESS"},
		{"start < end", "START_U3c_END"},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if actual := enumValToScreamingSnakeCase(tt.val); actual != tt.expected {
				t.Errorf("enumValToScreamingSnakeCase(%q) = %q; want %q", tt.val, actual, tt.expected)
			}
		})
	}
}
