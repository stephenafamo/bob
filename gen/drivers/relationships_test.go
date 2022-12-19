package drivers

import (
	"testing"

	"github.com/volatiletech/strmangle"
)

func TestJoinTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Pkey   []string
		Fkey   []string
		Should bool
	}{
		{Pkey: []string{"one", "two"}, Fkey: []string{"one", "two"}, Should: true},
		{Pkey: []string{"two", "one"}, Fkey: []string{"one", "two"}, Should: true},

		{Pkey: []string{"one"}, Fkey: []string{"one"}, Should: false},
		{Pkey: []string{"one", "two", "three"}, Fkey: []string{"one", "two"}, Should: false},
		{Pkey: []string{"one", "two", "three"}, Fkey: []string{"one", "two", "three"}, Should: false},
		{Pkey: []string{"one"}, Fkey: []string{"one", "two"}, Should: false},
		{Pkey: []string{"one", "two"}, Fkey: []string{"one"}, Should: false},
	}

	for i, test := range tests {
		var table Table

		table.PKey = &PrimaryKey{Columns: test.Pkey}
		for _, col := range strmangle.SetMerge(test.Pkey, test.Fkey) {
			table.Columns = append(table.Columns, Column{Name: col})
		}
		for _, k := range test.Fkey {
			table.FKeys = append(table.FKeys, ForeignKey{
				Constraint: Constraint{Columns: []string{k}},
			})
		}

		if IsJoinTable(table) != test.Should {
			t.Errorf("%d) want: %t, got: %t\nTest: %#v", i, test.Should, !test.Should, test)
		}
	}
}
