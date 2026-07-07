package drivers

import (
	"context"
	"testing"
)

// mockConstructor satisfies Constructor[any, any] for testing BuildDBInfo.
type mockConstructor struct {
	columns []Column
}

func (m mockConstructor) TablesInfo(_ context.Context, _ Filter) (TablesInfo, error) {
	return TablesInfo{{Key: "t", Schema: "public", Name: "t"}}, nil
}

func (m mockConstructor) TableDetails(_ context.Context, info TableInfo, _ ColumnFilter) (string, string, []Column, error) {
	cp := make([]Column, len(m.columns))
	copy(cp, m.columns)
	return info.Schema, info.Name, cp, nil
}

func (m mockConstructor) Comments(_ context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (m mockConstructor) Constraints(_ context.Context, _ ColumnFilter) (DBConstraints[any], error) {
	return DBConstraints[any]{
		PKs:     map[string]*Constraint[any]{},
		FKs:     map[string][]ForeignKey[any]{},
		Uniques: map[string][]Constraint[any]{},
		Checks:  map[string][]Check[any]{},
	}, nil
}

func (m mockConstructor) Indexes(_ context.Context) (DBIndexes[any], error) {
	return DBIndexes[any]{}, nil
}

func TestBuildDBInfoColumnOrder(t *testing.T) {
	t.Parallel()

	cols := []Column{
		{Name: "zebra"},
		{Name: "apple"},
		{Name: "mango"},
	}

	t.Run("ordinal preserves input order", func(t *testing.T) {
		t.Parallel()
		tables, err := BuildDBInfo[any](context.Background(), mockConstructor{cols}, 1, nil, nil, "ordinal")
		if err != nil {
			t.Fatal(err)
		}
		got := ColumnNames(tables[0].Columns)
		want := []string{"zebra", "apple", "mango"}
		for i, name := range want {
			if got[i] != name {
				t.Errorf("position %d: got %q, want %q", i, got[i], name)
			}
		}
	})

	t.Run("default (empty) preserves input order", func(t *testing.T) {
		t.Parallel()
		tables, err := BuildDBInfo[any](context.Background(), mockConstructor{cols}, 1, nil, nil, "")
		if err != nil {
			t.Fatal(err)
		}
		got := ColumnNames(tables[0].Columns)
		want := []string{"zebra", "apple", "mango"}
		for i, name := range want {
			if got[i] != name {
				t.Errorf("position %d: got %q, want %q", i, got[i], name)
			}
		}
	})

	t.Run("name sorts alphabetically", func(t *testing.T) {
		t.Parallel()
		tables, err := BuildDBInfo[any](context.Background(), mockConstructor{cols}, 1, nil, nil, "name")
		if err != nil {
			t.Fatal(err)
		}
		got := ColumnNames(tables[0].Columns)
		want := []string{"apple", "mango", "zebra"}
		for i, name := range want {
			if got[i] != name {
				t.Errorf("position %d: got %q, want %q", i, got[i], name)
			}
		}
	})
}

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
