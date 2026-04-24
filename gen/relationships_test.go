package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/orm"
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

func TestGetInverse(t *testing.T) {
	t.Parallel()

	// Self-referencing FK: tree.parent_record_id -> tree.id
	// The generator records two relationships on the "tree" table:
	//   - the forward (many-to-one) "tree_parent_id_fkey"
	//   - the reverse (one-to-many) "tree_parent_id_fkey__self_join_reverse"
	// GetInverse must return the *other* one, never the same relationship.
	selfFKForward := orm.Relationship{
		Name: "tree_parent_id_fkey",
		Sides: []orm.RelSide{{
			From:        "tree",
			To:          "tree",
			FromColumns: []string{"parent_record_id"},
			ToColumns:   []string{"id"},
		}},
	}
	selfFKReverse := orm.Relationship{
		Name: "tree_parent_id_fkey" + selfJoinSuffix,
		Sides: []orm.RelSide{{
			From:        "tree",
			To:          "tree",
			FromColumns: []string{"id"},
			ToColumns:   []string{"parent_record_id"},
		}},
	}

	// Ordinary FK: order_items.order_id -> orders.id
	// Forward (many-to-one) lives on order_items, reverse (one-to-many) lives on orders.
	// Both sides share the same Name, so GetInverse looks it up by name on the foreign table.
	orderFKForward := orm.Relationship{
		Name: "order_items_order_id_fkey",
		Sides: []orm.RelSide{{
			From:        "order_items",
			To:          "orders",
			FromColumns: []string{"order_id"},
			ToColumns:   []string{"id"},
		}},
	}
	orderFKReverse := orm.Relationship{
		Name: "order_items_order_id_fkey",
		Sides: []orm.RelSide{{
			From:        "orders",
			To:          "order_items",
			FromColumns: []string{"id"},
			ToColumns:   []string{"order_id"},
		}},
	}

	rels := Relationships{
		"tree":        {selfFKForward, selfFKReverse},
		"orders":      {orderFKReverse},
		"order_items": {orderFKForward},
	}

	tests := []struct {
		name string
		in   orm.Relationship
		want orm.Relationship
	}{
		{
			name: "self-FK forward returns suffixed reverse, not itself",
			in:   selfFKForward,
			want: selfFKReverse,
		},
		{
			name: "self-FK reverse returns unsuffixed forward, not itself",
			in:   selfFKReverse,
			want: selfFKForward,
		},
		{
			name: "ordinary FK forward returns reverse on foreign table",
			in:   orderFKForward,
			want: orderFKReverse,
		},
		{
			name: "ordinary FK reverse returns forward on local table",
			in:   orderFKReverse,
			want: orderFKForward,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rels.GetInverse(tt.in)
			if got.Name != tt.want.Name {
				t.Fatalf("inverse name: want %q, got %q", tt.want.Name, got.Name)
			}
			if got.Local() != tt.want.Local() || got.Foreign() != tt.want.Foreign() {
				t.Fatalf("inverse sides: want %s->%s, got %s->%s",
					tt.want.Local(), tt.want.Foreign(),
					got.Local(), got.Foreign())
			}
			// Guard against the original bug: for self-FKs, the inverse must
			// not point back at the same relationship name as the input.
			if tt.in.Local() == tt.in.Foreign() && got.Name == tt.in.Name {
				t.Fatalf("self-FK inverse must differ from input %q, got identical name", tt.in.Name)
			}
		})
	}
}

func TestFilterGeneratedRelationships(t *testing.T) {
	t.Parallel()

	relationships := Relationships{
		"child": {
			{
				Name: "child_parent_fk",
				Sides: []orm.RelSide{{
					From:        "child",
					FromColumns: []string{"parent_id"},
					To:          "parent",
					ToColumns:   []string{"id"},
					Modify:      "from",
					ToUnique:    true,
				}},
			},
		},
		"parent": {
			{
				Name: "child_parent_fk",
				Sides: []orm.RelSide{{
					From:        "parent",
					FromColumns: []string{"id"},
					To:          "child",
					ToColumns:   []string{"parent_id"},
					Modify:      "to",
					ToUnique:    false,
				}},
			},
		},
	}
	aliases := drivers.Aliases{
		"child": {
			Relationships: map[string]string{
				"child_parent_fk": "Parent",
			},
		},
		"parent": {
			Relationships: map[string]string{
				"child_parent_fk": "Children",
			},
		},
	}

	t.Run("all is the default", func(t *testing.T) {
		t.Parallel()

		got, err := filterGeneratedRelationships(Config[any]{}, aliases, relationships)
		if err != nil {
			t.Fatal(err)
		}
		if len(got["child"]) != 1 || len(got["parent"]) != 1 {
			t.Fatalf("expected all relationships to remain, got %#v", got)
		}
	})

	t.Run("to_one drops to-many relationships", func(t *testing.T) {
		t.Parallel()

		got, err := filterGeneratedRelationships(Config[any]{
			RelationshipCodegen: RelationshipCodegen{Mode: relationshipCodegenModeToOne},
		}, aliases, relationships)
		if err != nil {
			t.Fatal(err)
		}
		if len(got["child"]) != 1 {
			t.Fatalf("expected to-one relationship to remain, got %#v", got["child"])
		}
		if len(got["parent"]) != 0 {
			t.Fatalf("expected to-many relationship to be dropped, got %#v", got["parent"])
		}
	})

	t.Run("to_one keeps allowlisted to-many relationships", func(t *testing.T) {
		t.Parallel()

		got, err := filterGeneratedRelationships(Config[any]{
			RelationshipCodegen: RelationshipCodegen{
				Mode: relationshipCodegenModeToOne,
				AllowToMany: map[string][]string{
					"parent": {"Children"},
				},
			},
		}, aliases, relationships)
		if err != nil {
			t.Fatal(err)
		}
		if len(got["child"]) != 1 || len(got["parent"]) != 1 {
			t.Fatalf("expected allowlisted to-many relationship to remain, got %#v", got)
		}
	})

	t.Run("invalid mode returns an error", func(t *testing.T) {
		t.Parallel()

		_, err := filterGeneratedRelationships(Config[any]{
			RelationshipCodegen: RelationshipCodegen{Mode: "invalid"},
		}, aliases, relationships)
		if err == nil {
			t.Fatal("expected an error")
		}
		if want := `unknown relationship_codegen.mode "invalid"`; err.Error() != want {
			t.Fatalf("got %q, want %q", err.Error(), want)
		}
	})
}
