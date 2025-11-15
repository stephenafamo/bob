package parser

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func TestParseQueryConfig_Batch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected drivers.QueryConfig
	}{
		{
			name:  "batch keyword",
			input: ":::batch",
			expected: drivers.QueryConfig{
				Batch: true,
			},
		},
		{
			name:  "batch with true",
			input: ":::true",
			expected: drivers.QueryConfig{
				Batch: true,
			},
		},
		{
			name:  "batch with yes",
			input: ":::yes",
			expected: drivers.QueryConfig{
				Batch: true,
			},
		},
		{
			name:  "batch with 1",
			input: ":::1",
			expected: drivers.QueryConfig{
				Batch: true,
			},
		},
		{
			name:  "batch with uppercase",
			input: ":::BATCH",
			expected: drivers.QueryConfig{
				Batch: true,
			},
		},
		{
			name:  "full config with batch",
			input: "*User:[]User:slice:batch",
			expected: drivers.QueryConfig{
				ResultTypeOne:     "*User",
				ResultTypeAll:     "[]User",
				ResultTransformer: "slice",
				Batch:             true,
			},
		},
		{
			name:  "partial config with batch (skip transformer)",
			input: "::slice:batch",
			expected: drivers.QueryConfig{
				ResultTransformer: "slice",
				Batch:             true,
			},
		},
		{
			name:  "batch false (no keyword)",
			input: ":::false",
			expected: drivers.QueryConfig{
				Batch: false,
			},
		},
		{
			name:  "batch false (empty 4th param)",
			input: ":::",
			expected: drivers.QueryConfig{
				Batch: false,
			},
		},
		{
			name:  "only first three params",
			input: "*User:[]User:slice",
			expected: drivers.QueryConfig{
				ResultTypeOne:     "*User",
				ResultTypeAll:     "[]User",
				ResultTransformer: "slice",
				Batch:             false,
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: drivers.QueryConfig{
				Batch: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseQueryConfig(tt.input)

			if result.ResultTypeOne != tt.expected.ResultTypeOne {
				t.Errorf("ResultTypeOne: got %q, want %q", result.ResultTypeOne, tt.expected.ResultTypeOne)
			}

			if result.ResultTypeAll != tt.expected.ResultTypeAll {
				t.Errorf("ResultTypeAll: got %q, want %q", result.ResultTypeAll, tt.expected.ResultTypeAll)
			}

			if result.ResultTransformer != tt.expected.ResultTransformer {
				t.Errorf("ResultTransformer: got %q, want %q", result.ResultTransformer, tt.expected.ResultTransformer)
			}

			if result.Batch != tt.expected.Batch {
				t.Errorf("Batch: got %v, want %v", result.Batch, tt.expected.Batch)
			}
		})
	}
}

func TestQueryConfig_Merge_Batch(t *testing.T) {
	tests := []struct {
		name     string
		base     drivers.QueryConfig
		other    drivers.QueryConfig
		expected drivers.QueryConfig
	}{
		{
			name: "merge batch true",
			base: drivers.QueryConfig{
				ResultTypeOne: "*User",
				Batch:         false,
			},
			other: drivers.QueryConfig{
				Batch: true,
			},
			expected: drivers.QueryConfig{
				ResultTypeOne: "*User",
				Batch:         true,
			},
		},
		{
			name: "merge keeps base batch when other is false",
			base: drivers.QueryConfig{
				Batch: true,
			},
			other: drivers.QueryConfig{
				Batch: false,
			},
			expected: drivers.QueryConfig{
				Batch: true, // base value kept because other.Batch is false
			},
		},
		{
			name: "merge all fields including batch",
			base: drivers.QueryConfig{},
			other: drivers.QueryConfig{
				ResultTypeOne:     "*User",
				ResultTypeAll:     "[]User",
				ResultTransformer: "slice",
				Batch:             true,
			},
			expected: drivers.QueryConfig{
				ResultTypeOne:     "*User",
				ResultTypeAll:     "[]User",
				ResultTransformer: "slice",
				Batch:             true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.base.Merge(tt.other)

			if result.ResultTypeOne != tt.expected.ResultTypeOne {
				t.Errorf("ResultTypeOne: got %q, want %q", result.ResultTypeOne, tt.expected.ResultTypeOne)
			}

			if result.ResultTypeAll != tt.expected.ResultTypeAll {
				t.Errorf("ResultTypeAll: got %q, want %q", result.ResultTypeAll, tt.expected.ResultTypeAll)
			}

			if result.ResultTransformer != tt.expected.ResultTransformer {
				t.Errorf("ResultTransformer: got %q, want %q", result.ResultTransformer, tt.expected.ResultTransformer)
			}

			if result.Batch != tt.expected.Batch {
				t.Errorf("Batch: got %v, want %v", result.Batch, tt.expected.Batch)
			}
		})
	}
}
