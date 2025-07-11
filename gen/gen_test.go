package gen

import (
	"strings"
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/orm"
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
					Name:       "domain",
					Type:       "string",
					DBType:     "text",
					DomainName: "domain name",
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
	})

	replacements := []Replace{
		{
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: "excellent.Type",
		},
		{
			Tables: []string{"named_table"},
			Match: drivers.Column{
				Name: "id",
			},
			Replace: "excellent.NamedType",
		},
		{
			Match: drivers.Column{
				Type:     "null.String",
				Nullable: true,
			},
			Replace: "int",
		},
		{
			Match: drivers.Column{
				DomainName: "domain name",
			},
			Replace: "contextInt",
		},
		{
			Match: drivers.Column{
				Name: "by_named",
			},
			Replace: "big.Int",
		},
		{
			Match: drivers.Column{
				Comment: "xid",
			},
			Replace: "xid.ID",
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

	if typ := tables[1].Columns[0].Type; typ != "excellent.NamedType" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[1].Columns[1].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}
}

func TestAliasClashes(t *testing.T) {
	type aliasClashTestCase struct {
		name                   string
		aliases                drivers.Aliases
		tables                 drivers.Tables[any, any]
		relationships          Relationships
		expectedErrorFragments []string
		expectError            bool
	}

	// Helper to create minimal table data
	makeTable := func(key string, colNames ...string) drivers.Table[any, any] {
		cols := make([]drivers.Column, len(colNames))
		for i, name := range colNames {
			cols[i] = drivers.Column{Name: name}
		}
		return drivers.Table[any, any]{Key: key, Columns: cols}
	}

	testCases := []aliasClashTestCase{
		// Table Alias Clashes
		{
			name: "Table Alias: UpSingular vs UpPlural (Same Table)",
			aliases: drivers.Aliases{
				"tests": {UpSingular: "Test", UpPlural: "Test"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"UpSingular 'Test' (table 'tests') conflicts with UpPlural 'Test' (table 'tests')"},
			expectError:            true,
		},
		{
			name: "Table Alias: UpSingular vs UpSingular (Different Tables)",
			aliases: drivers.Aliases{
				"tests1": {UpSingular: "MyItem"},
				"tests2": {UpSingular: "MyItem"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests1"), makeTable("tests2")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"UpSingular 'MyItem' used by table 'tests1' and table 'tests2'"},
			expectError:            true,
		},
		{
			name: "Table Alias: UpSingular (Table1) vs UpPlural (Table2)",
			aliases: drivers.Aliases{
				"tests1": {UpSingular: "Items"},
				"tests2": {UpPlural: "Items"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests1"), makeTable("tests2")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"UpSingular 'Items' (table 'tests1') conflicts with UpPlural 'Items' (table 'tests2')"},
			expectError:            true,
		},
		{
			name: "Table Alias: DownSingular vs DownPlural (Same Table)",
			aliases: drivers.Aliases{
				"tests": {DownSingular: "test", DownPlural: "test"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"DownSingular 'test' (table 'tests') conflicts with DownPlural 'test' (table 'tests')"},
			expectError:            true,
		},
		{
			name: "Table Alias: DownSingular vs DownSingular (Different Tables)",
			aliases: drivers.Aliases{
				"tests1": {DownSingular: "myItem"},
				"tests2": {DownSingular: "myItem"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests1"), makeTable("tests2")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"DownSingular 'myItem' used by table 'tests1' and table 'tests2'"},
			expectError:            true,
		},
		{
			name: "Table Alias: DownSingular (Table1) vs DownPlural (Table2)",
			aliases: drivers.Aliases{
				"tests1": {DownSingular: "items"},
				"tests2": {DownPlural: "items"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests1"), makeTable("tests2")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"DownSingular 'items' (table 'tests1') conflicts with DownPlural 'items' (table 'tests2')"},
			expectError:            true,
		},
		{
			name: "Table Alias: No Clash (Valid Aliases)",
			aliases: drivers.Aliases{
				"tests1": {UpSingular: "Apple", UpPlural: "Apples", DownSingular: "apple", DownPlural: "apples"},
				"tests2": {UpSingular: "Banana", UpPlural: "Bananas", DownSingular: "banana", DownPlural: "bananas"},
			},
			tables:                 drivers.Tables[any, any]{makeTable("tests1"), makeTable("tests2")},
			relationships:          Relationships{},
			expectedErrorFragments: nil,
			expectError:            false,
		},
		// Column Alias Clashes
		{
			name: "Column Alias: Generated Clash",
			aliases: drivers.Aliases{
				"items": {}, // No user-defined column aliases
			},
			tables:                 drivers.Tables[any, any]{makeTable("items", "item_name", "itemName")}, // item_name -> ItemName, itemName -> ItemName
			relationships:          Relationships{},
			expectedErrorFragments: []string{"alias clash in table 'items': column alias 'ItemName' is used by both column 'item_name' and column 'itemName'"},
			expectError:            true,
		},
		{
			name: "Column Alias: User-Defined Clash",
			aliases: drivers.Aliases{
				"items": {
					Columns: map[string]string{
						"col_a": "MyField",
						"col_b": "MyField",
					},
				},
			},
			tables:                 drivers.Tables[any, any]{makeTable("items", "col_a", "col_b")},
			relationships:          Relationships{},
			expectedErrorFragments: []string{"alias clash in table 'items': column alias 'MyField' is used by both column 'col_a' and column 'col_b'"},
			expectError:            true,
		},
		{
			name: "Column Alias: No Clash",
			aliases: drivers.Aliases{
				"items": {},
			},
			tables:                 drivers.Tables[any, any]{makeTable("items", "first_name", "last_name")},
			relationships:          Relationships{},
			expectedErrorFragments: nil,
			expectError:            false,
		},
		// Relationship Alias Clashes
		{
			name: "Relationship Alias: User-Defined Clash",
			aliases: drivers.Aliases{
				"users": {
					Relationships: map[string]string{
						"RelA": "PrimaryRel",
						"RelB": "PrimaryRel",
					},
				},
			},
			tables: drivers.Tables[any, any]{makeTable("users")},
			relationships: Relationships{
				"users": []orm.Relationship{
					{Name: "RelA"}, // This is the original relationship name from the schema or earlier processing
					{Name: "RelB"},
				},
			},
			expectedErrorFragments: []string{"alias clash in table 'users': relationship alias 'PrimaryRel' is used by both relationship 'RelA' and relationship 'RelB'"},
			expectError:            true,
		},
		{
			name: "Relationship Alias: Generated Clash (Simulated via User Aliases)",
			// We simulate a generated clash by providing user aliases that mimic what problematic generation would do.
			// initAliases itself doesn't call relAlias; it expects tableAlias.Relationships to be populated
			// (either by user config or by prior call to relAlias whose results are put into tableAlias.Relationships).
			// So, this test is valid for testing the clash detection part of initAliases.
			aliases: drivers.Aliases{
				"users": {
					Relationships: map[string]string{
						// These keys are the *original* relationship names.
						"home_address_rel": "Addresses", // User/generation sets alias to "Addresses"
						"work_address_rel": "Addresses", // User/generation sets alias to "Addresses" for a different original relationship
					},
				},
			},
			tables: drivers.Tables[any, any]{makeTable("users")},
			relationships: Relationships{
				"users": []orm.Relationship{ // These are the relationships as they would exist before alias assignment for this table
					{Name: "home_address_rel"},
					{Name: "work_address_rel"},
				},
			},
			expectedErrorFragments: []string{"alias clash in table 'users': relationship alias 'Addresses' is used by both relationship 'home_address_rel' and relationship 'work_address_rel'"},
			expectError:            true,
		},
		{
			name: "Relationship Alias: No Clash",
			aliases: drivers.Aliases{
				"users": {
					Relationships: map[string]string{
						"OrdersRel":  "UserOrders",
						"ProfileRel": "UserProfile",
					},
				},
			},
			tables: drivers.Tables[any, any]{makeTable("users")},
			relationships: Relationships{
				"users": []orm.Relationship{
					{Name: "OrdersRel"},  // Original name maps to "UserOrders"
					{Name: "ProfileRel"}, // Original name maps to "UserProfile"
				},
			},
			expectedErrorFragments: nil,
			expectError:            false,
		},
		{
			name: "Relationship Alias: No Clash - Generated (Default behavior if relAlias produces unique names)",
			// This test assumes that if no user aliases are provided, the names generated by a prior
			// (hypothetical, for this specific test's scope) call to relAlias and placed into
			// tableAlias.Relationships are unique. initAliases will then just use them.
			aliases: drivers.Aliases{
				"users": {
					// tableAlias.Relationships would be pre-populated by relAlias like:
					// "home_address_rel": "HomeAddress",
					// "work_address_rel": "WorkAddress",
					// We set it up directly here.
					Relationships: map[string]string{
						"home_address_rel": "HomeAddress",
						"work_address_rel": "WorkAddress",
					},
				},
			},
			tables: drivers.Tables[any, any]{makeTable("users")},
			relationships: Relationships{
				"users": []orm.Relationship{
					{Name: "home_address_rel"},
					{Name: "work_address_rel"},
				},
			},
			expectedErrorFragments: nil,
			expectError:            false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure Aliases map is initialized if nil for a test case
			if tc.aliases == nil {
				tc.aliases = make(drivers.Aliases)
			}
			// Ensure Relationships map is initialized if nil
			if tc.relationships == nil {
				tc.relationships = make(Relationships)
			}

			// For relationship tests, tableAlias.Relationships would typically be populated
			// by a call to relAlias() if not specified by the user.
			// Here, we are testing initAliases's clash detection, so if tc.aliases[tableKey].Relationships
			// is set, initAliases will use it. If it's not set, initAliases iterates over tc.relationships[tableKey]
			// and tries to fill it (potentially using computed names, though relAlias is not called within initAliases).
			// The test cases for relationships directly set tc.aliases[tableKey].Relationships to simulate
			// the state *after* user definition or generation by relAlias would have occurred.

			// 1. Populate aliases using initAliases (mutates tc.aliases)
			initAliases(tc.aliases, tc.tables, tc.relationships)

			// 2. Validate the populated aliases
			errors := validateAliases(tc.aliases, tc.tables, tc.relationships)

			if tc.expectError {
				if len(errors) == 0 {
					t.Fatalf("expected errors but got none")
				}
				var allErrorsStr strings.Builder
				for _, err := range errors {
					allErrorsStr.WriteString(err.Error() + "\n")
				}
				fullErrorMsg := allErrorsStr.String()

				if len(tc.expectedErrorFragments) == 0 {
					t.Error("expectError is true, but no expectedErrorFragments were provided")
				}

				for _, fragment := range tc.expectedErrorFragments {
					if !strings.Contains(fullErrorMsg, fragment) {
						t.Errorf("expected error message to contain %q, but got: %s", fragment, fullErrorMsg)
					}
				}
			} else {
				if len(errors) > 0 {
					var allErrorsStr strings.Builder
					for _, err := range errors {
						allErrorsStr.WriteString(err.Error() + "\n")
					}
					t.Errorf("expected no errors, but got: %v", allErrorsStr.String())
				}
			}
		})
	}
}
