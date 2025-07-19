package drivers

import (
	"slices"
	"testing"
)

func Test_isRegexPattern(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		delimiter string
		expected  bool
	}{
		{
			name:      "valid",
			pattern:   "/a/",
			delimiter: "/",
			expected:  true,
		},
		{
			name:      "not closed",
			pattern:   "/a",
			delimiter: "/",
			expected:  false,
		},
		{
			name:      "not delimited",
			pattern:   "a",
			delimiter: "/",
			expected:  false,
		},
	}
	for _, tt := range tests {
		if actual := isRegexPattern(tt.pattern, tt.delimiter); actual != tt.expected {
			t.Errorf("isRegexPattern() = %v, want %v", actual, tt.expected)
		}
	}
}

func TestClassifyPatterns(t *testing.T) {
	tests := []struct {
		name                   string
		patterns               []string
		expectedStringPatterns []string
		expectedRegexPatterns  []string
	}{
		{
			name: "valid regex patterns",
			patterns: []string{
				"/regex/",
				"not regex",
			},
			expectedStringPatterns: []string{"not regex"},
			expectedRegexPatterns:  []string{"regex"},
		},
		{
			name: "invalid regex patterns",
			patterns: []string{
				"/bad regex",
			},
			expectedStringPatterns: []string{"/bad regex"},
			expectedRegexPatterns:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualStringPatterns, actualRegexPatterns := ClassifyPatterns(tt.patterns)
			if !slices.Equal(actualStringPatterns, tt.expectedStringPatterns) {
				t.Errorf("ClassifyPatterns() got = %v, want %v", actualStringPatterns, tt.expectedStringPatterns)
			}
			if !slices.Equal(actualRegexPatterns, tt.expectedRegexPatterns) {
				t.Errorf("ClassifyPatterns() got = %v, want %v", actualRegexPatterns, tt.expectedRegexPatterns)
			}
		})
	}
}
