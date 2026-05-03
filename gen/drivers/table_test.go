package drivers

import (
	"testing"

	"github.com/stephenafamo/bob/orm"
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

func TestRelIsRequired(t *testing.T) {
	t.Parallel()

	table := Table[any, any]{
		Columns: []Column{
			{Name: "required_one", Nullable: false},
			{Name: "required_two", Nullable: false},
			{Name: "optional_one", Nullable: true},
		},
	}

	tests := []struct {
		name string
		rel  orm.Relationship
		want bool
	}{
		{
			name: "never required",
			rel: orm.Relationship{
				NeverRequired: true,
				Sides: []orm.RelSide{{
					Modify:      "from",
					FromColumns: []string{"required_one"},
				}},
			},
			want: false,
		},
		{
			name: "modify to",
			rel: orm.Relationship{
				Sides: []orm.RelSide{{
					Modify:      "to",
					FromColumns: []string{"required_one"},
				}},
			},
			want: false,
		},
		{
			name: "empty from columns",
			rel: orm.Relationship{
				Sides: []orm.RelSide{{
					Modify: "from",
				}},
			},
			want: false,
		},
		{
			name: "single required column",
			rel: orm.Relationship{
				Sides: []orm.RelSide{{
					Modify:      "from",
					FromColumns: []string{"required_one"},
				}},
			},
			want: true,
		},
		{
			name: "single optional column",
			rel: orm.Relationship{
				Sides: []orm.RelSide{{
					Modify:      "from",
					FromColumns: []string{"optional_one"},
				}},
			},
			want: false,
		},
		{
			name: "all composite columns required",
			rel: orm.Relationship{
				Sides: []orm.RelSide{{
					Modify:      "from",
					FromColumns: []string{"required_one", "required_two"},
				}},
			},
			want: true,
		},
		{
			name: "mixed composite columns optional",
			rel: orm.Relationship{
				Sides: []orm.RelSide{{
					Modify:      "from",
					FromColumns: []string{"required_one", "optional_one"},
				}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := table.RelIsRequired(tt.rel); got != tt.want {
				t.Fatalf("RelIsRequired() = %t, want %t", got, tt.want)
			}
		})
	}
}
