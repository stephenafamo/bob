package gen

import (
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob/gen/drivers"
)

func TestColumnFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   ColumnFilter
		expected bool
	}{
		{
			name:     "empty",
			filter:   ColumnFilter{},
			expected: true,
		},
		{
			name:     "not empty",
			filter:   ColumnFilter{Name: omit.From("id")},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.filter.IsEmpty()
			if actual != tt.expected {
				t.Errorf("ColumnFilter.IsEmpty() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestColumnFilter_Matches(t *testing.T) {
	tests := []struct {
		name     string
		filter   ColumnFilter
		column   drivers.Column
		expected bool
	}{
		{
			name:     "empty filter doesn't match anything",
			filter:   ColumnFilter{},
			column:   drivers.Column{Name: "id"},
			expected: false,
		},
		{
			name:     "names do not match",
			filter:   ColumnFilter{Name: omit.From("not_id")},
			column:   drivers.Column{Name: "id"},
			expected: false,
		},
		{
			name:     "names match case-insensitively",
			filter:   ColumnFilter{Name: omit.From("ID")},
			column:   drivers.Column{Name: "id"},
			expected: true,
		},
		{
			name:     "names match with regex",
			filter:   ColumnFilter{Name: omit.From("/id$/")},
			column:   drivers.Column{Name: "author_id"},
			expected: true,
		},
		{
			name:     "regex is always case insensitive",
			filter:   ColumnFilter{Name: omit.From("/ID$/")},
			column:   drivers.Column{Name: "author_id"},
			expected: true,
		},
		{
			name:     "types match case-insensitively",
			filter:   ColumnFilter{Type: omit.From("INTEGER")},
			column:   drivers.Column{Type: "integer"},
			expected: true,
		},
		{
			name:     "db types match case-insensitively",
			filter:   ColumnFilter{DBType: omit.From("int")},
			column:   drivers.Column{DBType: "INT"},
			expected: true,
		},
		{
			name:     "domain names match case-insensitively",
			filter:   ColumnFilter{DomainName: omit.From("email")},
			column:   drivers.Column{DomainName: "EMAIL"},
			expected: true,
		},
		{
			name:     "default values match case-insensitively",
			filter:   ColumnFilter{Default: omit.From("null")},
			column:   drivers.Column{Default: "NULL"},
			expected: true,
		},
		{
			name:     "comments match case-insensitively",
			filter:   ColumnFilter{Comment: omit.From("primary key")},
			column:   drivers.Column{Comment: "PRIMARY KEY"},
			expected: true,
		},
		{
			name:     "generated matches exactly",
			filter:   ColumnFilter{Generated: omit.From(false)},
			column:   drivers.Column{Generated: false},
			expected: true,
		},
		{
			name:     "generated does not match",
			filter:   ColumnFilter{Generated: omit.From(false)},
			column:   drivers.Column{Generated: true},
			expected: false,
		},
		{
			name:     "autoincr matches exactly",
			filter:   ColumnFilter{AutoIncr: omit.From(false)},
			column:   drivers.Column{AutoIncr: false},
			expected: true,
		},
		{
			name:     "autoincr does not match",
			filter:   ColumnFilter{AutoIncr: omit.From(false)},
			column:   drivers.Column{AutoIncr: true},
			expected: false,
		},
		{
			name:     "nullable matches exactly",
			filter:   ColumnFilter{Nullable: omit.From(true)},
			column:   drivers.Column{Nullable: true},
			expected: true,
		},
		{
			name:     "nullable does not match",
			filter:   ColumnFilter{Nullable: omit.From(true)},
			column:   drivers.Column{Nullable: false},
			expected: false,
		},
		{
			name: "filters are combined with AND",
			filter: ColumnFilter{
				Name: omit.From("id"),
				Type: omit.From("wrong_type"),
			},
			column: drivers.Column{
				Name: "id",
				Type: "integer",
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.filter.Matches(tt.column); actual != tt.expected {
				t.Errorf("ColumnFilter.Matches() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
