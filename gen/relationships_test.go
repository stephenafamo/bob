package gen

import (
	"testing"

	"github.com/twitter-payments/bob/gen/drivers"
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
		var table drivers.Table[any, any]

		table.Constraints.Primary = &drivers.Constraint[any]{Columns: test.Pkey}
		for _, col := range strmangle.SetMerge(test.Pkey, test.Fkey) {
			table.Columns = append(table.Columns, drivers.Column{Name: col})
		}
		for _, k := range test.Fkey {
			table.Constraints.Foreign = append(
				table.Constraints.Foreign,
				drivers.ForeignKey[any]{
					Constraint: drivers.Constraint[any]{Columns: []string{k}},
				},
			)
		}

		if table.IsJoinTable() != test.Should {
			t.Errorf("%d) want: %t, got: %t\nTest: %#v", i, test.Should, !test.Should, test)
		}
	}
}
