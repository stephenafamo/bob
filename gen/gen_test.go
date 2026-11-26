package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

func TestProcessTypeReplacements(t *testing.T) {
	tables := drivers.Tables[any, any]{
		{
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     "null.String",
					DBType:   "text",
					Default:  "some db nonsense",
					Nullable: true,
				},
				{
					Name:       "user_email",
					Type:       "string",
					DBType:     "text",
					DomainName: "email",
				},
				{
					Name:     "by_named",
					Type:     "int",
					DBType:   "numeric",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "by_comment",
					Type:     "string",
					DBType:   "text",
					Default:  "some db nonsense",
					Nullable: false,
					Comment:  "xid",
				},
				{
					Name:     "author_id",
					Type:     "int",
					DBType:   "integer",
					Nullable: true,
				},
			},
		},
		{
			Key: "named_table",
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "by_comment",
					Type:     "string",
					DBType:   "text",
					Default:  "some db nonsense",
					Nullable: false,
					Comment:  "xid",
				},
			},
		},
		{
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					AutoIncr: true,
					Nullable: false,
				},
			},
		},
	}

	types := drivers.Types{}
	types.RegisterAll(map[string]drivers.Type{
		"excellent.Type": {
			Imports: []string{`"rock.com/excellent"`},
		},
		"excellent.NamedType": {
			Imports: []string{`"rock.com/excellent-name"`},
		},
		"int": {
			Imports: []string{`"context"`},
		},
		"contextInt": {
			Imports: []string{`"contextual"`},
		},
		"big.Int": {
			Imports: []string{`"math/big"`},
		},
		"xid.ID": {
			Imports: []string{`"github.com/rs/xid"`},
		},
		"fk.ID": {
			Imports: []string{`"github.com/fk"`},
		},
		"pk.ID": {
			Imports: []string{`"github.com/pk"`},
		},
	})

	replacements := []Replace{
		{
			Match: ColumnFilter{
				DBType: internal.Pointer("SERIAL"),
			},
			Replace: "excellent.Type",
		},
		{
			Tables: []string{"named_table"},
			Match: ColumnFilter{
				Name: internal.Pointer("id"),
			},
			Replace: "excellent.NamedType",
		},
		{
			Match: ColumnFilter{
				Type:     internal.Pointer("null.String"),
				Nullable: internal.Pointer(true),
			},
			Replace: "int",
		},
		{
			Match: ColumnFilter{
				DomainName: internal.Pointer("EMAIL"),
			},
			Replace: "contextInt",
		},
		{
			Match: ColumnFilter{
				Name: internal.Pointer("by_named"),
			},
			Replace: "big.Int",
		},
		{
			Match: ColumnFilter{
				Comment: internal.Pointer("xid"),
			},
			Replace: "xid.ID",
		},
		{
			Match: ColumnFilter{
				Name: internal.Pointer("/_id$/"),
			},
			Replace: "fk.ID",
		},
		{
			Match: ColumnFilter{
				Name:     internal.Pointer("id"),
				AutoIncr: internal.Pointer(true),
			},
			Replace: "pk.ID",
		},
	}

	processTypeReplacements(types, replacements, tables)

	if typ := tables[0].Columns[0].Type; typ != "excellent.Type" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[1].Type; typ != "int" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[2].Type; typ != "contextInt" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[3].Type; typ != "big.Int" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[4].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[5].Type; typ != "fk.ID" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[1].Columns[0].Type; typ != "excellent.NamedType" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[1].Columns[1].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[2].Columns[0].Type; typ != "pk.ID" {
		t.Error("type was wrong:", typ)
	}
}

func TestProcessTypeReplacements_ForeignKeyInference(t *testing.T) {
	t.Run("composite_fk_from_issue_634", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key:  "tax",
				Name: "tax",
				Columns: []drivers.Column{
					{Name: "id", Type: "string", DBType: "varchar(255)"},
					{Name: "cohort_id", Type: "int64", DBType: "bigint"},
				},
			},
			{
				Key:  "user_tax",
				Name: "user_tax",
				Columns: []drivers.Column{
					{Name: "id", Type: "string", DBType: "varchar(255)"},
					{Name: "cohort_id", Type: "int64", DBType: "bigint"},
					{Name: "tax_id", Type: "string", DBType: "text", Nullable: true},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Name: "fk_user_tax", Columns: []string{"cohort_id", "tax_id"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"cohort_id", "id"},
						},
					},
				},
			},
		}

		replacements := []Replace{
			{
				Tables: []string{"tax"},
				Match: ColumnFilter{
					Name: internal.Pointer("id"),
				},
				Replace: "typeid.TypeID",
			},
		}

		processTypeReplacements(drivers.Types{}, replacements, tables)

		if got := tables[0].Columns[0].Type; got != "typeid.TypeID" {
			t.Errorf("tax.id: got %q, want %q", got, "typeid.TypeID")
		}
		if got := tables[1].Columns[0].Type; got != "string" {
			t.Errorf("user_tax.id: got %q, want %q", got, "string")
		}
		if got := tables[1].Columns[1].Type; got != "int64" {
			t.Errorf("user_tax.cohort_id: got %q, want %q", got, "int64")
		}
		if got := tables[1].Columns[2].Type; got != "typeid.TypeID" {
			t.Errorf("user_tax.tax_id: got %q, want %q", got, "typeid.TypeID")
		}
	})

	t.Run("chained_fks_reverse_table_order", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key: "grandchild",
				Columns: []drivers.Column{
					{Name: "child_id", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"child_id"}},
							ForeignTable:   "child",
							ForeignColumns: []string{"parent_id"},
						},
					},
				},
			},
			{
				Key: "child",
				Columns: []drivers.Column{
					{Name: "parent_id", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"parent_id"}},
							ForeignTable:   "parent",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
			{
				Key: "parent",
				Columns: []drivers.Column{
					{Name: "id", Type: "string"},
				},
			},
		}

		replacements := []Replace{
			{
				Tables:  []string{"parent"},
				Match:   ColumnFilter{Name: internal.Pointer("id")},
				Replace: "custom.ID",
			},
		}

		processTypeReplacements(drivers.Types{}, replacements, tables)

		if got := tables[1].Columns[0].Type; got != "custom.ID" {
			t.Errorf("child.parent_id: got %q, want %q", got, "custom.ID")
		}
		if got := tables[0].Columns[0].Type; got != "custom.ID" {
			t.Errorf("grandchild.child_id: got %q, want %q", got, "custom.ID")
		}
	})

	t.Run("self_referential_and_circular_fks", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key: "employees",
				Columns: []drivers.Column{
					{Name: "id", Type: "string"},
					{Name: "manager_id", Type: "string", Nullable: true},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"manager_id"}},
							ForeignTable:   "employees",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
			{
				Key: "node_a",
				Columns: []drivers.Column{
					{Name: "id", Type: "string"},
					{Name: "b_id", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"b_id"}},
							ForeignTable:   "node_b",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
			{
				Key: "node_b",
				Columns: []drivers.Column{
					{Name: "id", Type: "string"},
					{Name: "a_id", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"a_id"}},
							ForeignTable:   "node_a",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
			{
				Key: "cycle_a",
				Columns: []drivers.Column{
					{Name: "x", Type: "int"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"x"}},
							ForeignTable:   "cycle_b",
							ForeignColumns: []string{"y"},
						},
					},
				},
			},
			{
				Key: "cycle_b",
				Columns: []drivers.Column{
					{Name: "y", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"y"}},
							ForeignTable:   "cycle_a",
							ForeignColumns: []string{"x"},
						},
					},
				},
			},
		}

		replacements := []Replace{
			{
				Tables:  []string{"employees"},
				Match:   ColumnFilter{Name: internal.Pointer("id")},
				Replace: "custom.EmpID",
			},
			{
				Tables:  []string{"node_a"},
				Match:   ColumnFilter{Name: internal.Pointer("id")},
				Replace: "custom.NodeAID",
			},
		}

		processTypeReplacements(drivers.Types{}, replacements, tables)

		if got := tables[0].Columns[1].Type; got != "custom.EmpID" {
			t.Errorf("employees.manager_id: got %q, want %q", got, "custom.EmpID")
		}
		if got := tables[2].Columns[1].Type; got != "custom.NodeAID" {
			t.Errorf("node_b.a_id: got %q, want %q", got, "custom.NodeAID")
		}
	})

	t.Run("broad_rule_then_specific_pk_rule", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key: "tax",
				Columns: []drivers.Column{
					{Name: "id", Type: "string", DBType: "uuid"},
				},
			},
			{
				Key: "user_tax",
				Columns: []drivers.Column{
					{Name: "tax_id", Type: "string", DBType: "uuid"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"tax_id"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
			{
				Key: "other",
				Columns: []drivers.Column{
					{Name: "val", Type: "string", DBType: "uuid"},
				},
			},
		}

		replacements := []Replace{
			{
				Match:   ColumnFilter{DBType: internal.Pointer("uuid")},
				Replace: "uuid.UUID",
			},
			{
				Tables:  []string{"tax"},
				Match:   ColumnFilter{Name: internal.Pointer("id")},
				Replace: "ids.TaxID",
			},
		}

		processTypeReplacements(drivers.Types{}, replacements, tables)

		if got := tables[0].Columns[0].Type; got != "ids.TaxID" {
			t.Errorf("tax.id: got %q, want %q", got, "ids.TaxID")
		}
		if got := tables[1].Columns[0].Type; got != "ids.TaxID" {
			t.Errorf("user_tax.tax_id: got %q, want %q", got, "ids.TaxID")
		}
		if got := tables[2].Columns[0].Type; got != "uuid.UUID" {
			t.Errorf("other.val: got %q, want %q", got, "uuid.UUID")
		}
	})

	t.Run("explicit_fk_override_after_pk_rule", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key: "tax",
				Columns: []drivers.Column{
					{Name: "id", Type: "string"},
				},
			},
			{
				Key: "user_tax",
				Columns: []drivers.Column{
					{Name: "tax_id", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"tax_id"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
		}

		replacements := []Replace{
			{
				Tables:  []string{"tax"},
				Match:   ColumnFilter{Name: internal.Pointer("id")},
				Replace: "ids.TaxID",
			},
			{
				Tables:  []string{"user_tax"},
				Match:   ColumnFilter{Name: internal.Pointer("tax_id")},
				Replace: "ids.CustomTaxID",
			},
		}

		processTypeReplacements(drivers.Types{}, replacements, tables)

		if got := tables[0].Columns[0].Type; got != "ids.TaxID" {
			t.Errorf("tax.id: got %q, want %q", got, "ids.TaxID")
		}
		if got := tables[1].Columns[0].Type; got != "ids.CustomTaxID" {
			t.Errorf("user_tax.tax_id: got %q, want %q", got, "ids.CustomTaxID")
		}
	})

	t.Run("unreplaced_fk_type_mismatch", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key: "grandchild",
				Columns: []drivers.Column{
					{Name: "child_id", Type: "int64"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"child_id"}},
							ForeignTable:   "child",
							ForeignColumns: []string{"parent_id"},
						},
					},
				},
			},
			{
				Key: "child",
				Columns: []drivers.Column{
					{Name: "parent_id", Type: "int32"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"parent_id"}},
							ForeignTable:   "parent",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
			{
				Key: "parent",
				Columns: []drivers.Column{
					{Name: "id", Type: "uint32"},
				},
			},
		}

		processTypeReplacements(drivers.Types{}, nil, tables)

		if got := tables[1].Columns[0].Type; got != "uint32" {
			t.Errorf("child.parent_id: got %q, want %q", got, "uint32")
		}
		if got := tables[0].Columns[0].Type; got != "uint32" {
			t.Errorf("grandchild.child_id: got %q, want %q", got, "uint32")
		}
	})

	t.Run("missing_foreign_table_and_mismatched_columns", func(t *testing.T) {
		tables := drivers.Tables[any, any]{
			{
				Key: "tax",
				Columns: []drivers.Column{
					{Name: "id", Type: "string"},
				},
			},
			{
				Key: "user_tax",
				Columns: []drivers.Column{
					{Name: "missing_table_fk", Type: "string"},
					{Name: "missing_col_fk", Type: "string"},
					{Name: "mismatched_len_fk", Type: "string"},
					{Name: "tax_id", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"missing_table_fk"}},
							ForeignTable:   "does_not_exist",
							ForeignColumns: []string{"id"},
						},
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"missing_col_fk"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"does_not_exist"},
						},
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"mismatched_len_fk"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"id", "extra"},
						},
						{
							Constraint:     drivers.Constraint[any]{Columns: []string{"tax_id"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
		}

		replacements := []Replace{
			{
				Tables:  []string{"tax"},
				Match:   ColumnFilter{Name: internal.Pointer("id")},
				Replace: "typeid.TypeID",
			},
		}

		processTypeReplacements(drivers.Types{}, replacements, tables)

		if got := tables[1].Columns[0].Type; got != "string" {
			t.Errorf("missing_table_fk: got %q, want %q", got, "string")
		}
		if got := tables[1].Columns[1].Type; got != "string" {
			t.Errorf("missing_col_fk: got %q, want %q", got, "string")
		}
		if got := tables[1].Columns[2].Type; got != "string" {
			t.Errorf("mismatched_len_fk: got %q, want %q", got, "string")
		}
		if got := tables[1].Columns[3].Type; got != "typeid.TypeID" {
			t.Errorf("tax_id: got %q, want %q", got, "typeid.TypeID")
		}
	})
}
