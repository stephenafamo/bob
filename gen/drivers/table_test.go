package drivers

import (
	"testing"
)

func TestGetTable(t *testing.T) {
	t.Parallel()

	tables := Tables[any, any]{
		{Key: "one"},
	}

	tbl := tables.Get("one")

	if tbl.Key != "one" {
		t.Error("didn't get column")
	}
}

func TestGetTableMissing(t *testing.T) {
	t.Parallel()

	tables := Tables[any, any]{
		{Key: "one"},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected a panic failure")
		}
	}()

	tables.Get("missing")
}

func TestGetColumn(t *testing.T) {
	t.Parallel()

	table := Table[any, any]{
		Columns: []Column{
			{Name: "one"},
		},
	}

	c := table.GetColumn("one")

	if c.Name != "one" {
		t.Error("didn't get column")
	}
}

func TestGetColumnMissing(t *testing.T) {
	t.Parallel()

	table := Table[any, any]{
		Columns: []Column{
			{Name: "one"},
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected a panic failure")
		}
	}()

	table.GetColumn("missing")
}

func TestCanSoftDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Can     bool
		Columns []Column
	}{
		{true, []Column{
			{Name: "deleted_at", Type: "null.Time"},
		}},
		{false, []Column{
			{Name: "deleted_at", Type: "time.Time"},
		}},
		{false, []Column{
			{Name: "deleted_at", Type: "int"},
		}},
		{false, nil},
	}

	for i, test := range tests {
		table := Table[any, any]{
			Columns: test.Columns,
		}

		if got := table.CanSoftDelete("deleted_at"); got != test.Can {
			t.Errorf("%d) wrong: %t", i, got)
		}
	}
}
