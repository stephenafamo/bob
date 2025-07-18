package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func Test_matchColumn(t *testing.T) {
	tests := []struct {
		name     string
		this     drivers.Column
		other    drivers.Column
		expected bool
	}{
		{
			name:     "names do not match",
			this:     drivers.Column{Name: "id"},
			other:    drivers.Column{Name: "not_id"},
			expected: false,
		},
		{
			name:     "names match case-insensitively",
			this:     drivers.Column{Name: "id"},
			other:    drivers.Column{Name: "ID"},
			expected: true,
		},
		{
			name:     "names match with regex",
			this:     drivers.Column{Name: "author_id"},
			other:    drivers.Column{Name: "/id$/"},
			expected: true,
		},
		{
			name:     "regex is always case insensitive",
			this:     drivers.Column{Name: "author_id"},
			other:    drivers.Column{Name: "/ID$/"},
			expected: true,
		},
		{
			name:     "types match case-insensitively",
			this:     drivers.Column{Type: "integer"},
			other:    drivers.Column{Type: "INTEGER"},
			expected: true,
		},
		{
			name:     "types match case-insensitively",
			this:     drivers.Column{Type: "integer"},
			other:    drivers.Column{Type: "INTEGER"},
			expected: true,
		},
		{
			name:     "db types match case-insensitively",
			this:     drivers.Column{DBType: "int"},
			other:    drivers.Column{DBType: "INT"},
			expected: true,
		},
		{
			name:     "domain names match case-insensitively",
			this:     drivers.Column{DomainName: "email"},
			other:    drivers.Column{DomainName: "EMAIL"},
			expected: true,
		},
		{
			name:     "comments do not match",
			this:     drivers.Column{Comment: "primary key"},
			other:    drivers.Column{Comment: "not primary key"},
			expected: false,
		},
		{
			name:     "generated is only checked if something else matched",
			this:     drivers.Column{Name: "full_name", Generated: false},
			other:    drivers.Column{Name: "full_name", Generated: true},
			expected: false,
		},
		{
			name:     "generated is not checked if nothing else matched",
			this:     drivers.Column{Generated: true},
			other:    drivers.Column{Generated: true},
			expected: false,
		},
		{
			name:     "generated should match",
			this:     drivers.Column{Name: "full_name", Generated: true},
			other:    drivers.Column{Name: "full_name", Generated: true},
			expected: true,
		},
		{
			name:     "autoincr is only checked if something else matched",
			this:     drivers.Column{Name: "id", AutoIncr: false},
			other:    drivers.Column{Name: "id", AutoIncr: true},
			expected: false,
		},
		{
			name:     "autoincr is not checked if nothing else matched",
			this:     drivers.Column{AutoIncr: true},
			other:    drivers.Column{AutoIncr: true},
			expected: false,
		},
		{
			name:     "autoincr should match",
			this:     drivers.Column{Name: "id", AutoIncr: true},
			other:    drivers.Column{Name: "id", AutoIncr: true},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := matchColumn(tt.this, tt.other); actual != tt.expected {
				t.Errorf("matchColumn() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
