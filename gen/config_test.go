package gen

import (
	"reflect"
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func TestConvertAliases(t *testing.T) {
	t.Parallel()

	var intf any = map[string]any{
		"tables": map[string]any{
			"table_name": map[string]any{
				"up_plural":     "a",
				"up_singular":   "b",
				"down_plural":   "c",
				"down_singular": "d",

				"columns": map[string]any{
					"a": "b",
				},
				"relationships": map[string]any{
					"ib_fk_1": "b",
				},
			},
		},
	}

	aliases := ConvertAliases(intf)

	if len(aliases.Tables) != 1 {
		t.Fatalf("should have one table alias: %#v", aliases.Tables)
	}

	table := aliases.Tables["table_name"]
	if table.UpPlural != "a" {
		t.Error("value was wrong:", table.UpPlural)
	}
	if table.UpSingular != "b" {
		t.Error("value was wrong:", table.UpSingular)
	}
	if table.DownPlural != "c" {
		t.Error("value was wrong:", table.DownPlural)
	}
	if table.DownSingular != "d" {
		t.Error("value was wrong:", table.DownSingular)
	}

	if len(table.Columns) != 1 {
		t.Error("should have one column")
	}

	if table.Columns["a"] != "b" {
		t.Error("column alias was wrong")
	}

	if len(aliases.Tables) != 1 {
		t.Fatal("should have one relationship alias")
	}

	if table.Relationships["ib_fk_1"] != "b" {
		t.Error("value was wrong:", table.Relationships["ib_fk_1"])
	}
}

func TestConvertAliasesAltSyntax(t *testing.T) {
	t.Parallel()

	var intf any = map[string]any{
		"tables": []any{
			map[string]any{
				"name":          "table_name",
				"up_plural":     "a",
				"up_singular":   "b",
				"down_plural":   "c",
				"down_singular": "d",

				"columns": []any{
					map[string]any{
						"name":  "a",
						"alias": "b",
					},
				},
				"relationships": []any{
					map[string]any{
						"name":  "ib_fk_1",
						"alias": "b",
					},
				},
			},
		},
	}

	aliases := ConvertAliases(intf)

	if len(aliases.Tables) != 1 {
		t.Fatalf("should have one table alias: %#v", aliases.Tables)
	}

	table := aliases.Tables["table_name"]
	if table.UpPlural != "a" {
		t.Error("value was wrong:", table.UpPlural)
	}
	if table.UpSingular != "b" {
		t.Error("value was wrong:", table.UpSingular)
	}
	if table.DownPlural != "c" {
		t.Error("value was wrong:", table.DownPlural)
	}
	if table.DownSingular != "d" {
		t.Error("value was wrong:", table.DownSingular)
	}

	if len(table.Columns) != 1 {
		t.Error("should have one column")
	}

	if table.Columns["a"] != "b" {
		t.Error("column alias was wrong")
	}

	if len(aliases.Tables) != 1 {
		t.Fatal("should have one relationship alias")
	}

	if table.Relationships["ib_fk_1"] != "b" {
		t.Error("value was wrong:", table.Relationships["ib_fk_1"])
	}
}

func columnWithImports(t *testing.T, c map[string]any, pkgs ...string) map[string]any {
	t.Helper()

	// first make a copy of the map
	c2 := make(map[string]any, len(c))
	for k, v := range c {
		c2[k] = v
	}

	c2Imports, _ := c2["imports"].([]string)
	c2Imports = append(c2Imports, pkgs...)
	c2["imports"] = c2Imports

	return c2
}

func TestConvertTypeReplace(t *testing.T) {
	t.Parallel()

	fullColumn := map[string]any{
		"name":           "a",
		"type":           "b",
		"db_type":        "c",
		"udt_name":       "d",
		"full_db_type":   "e",
		"arr_type":       "f",
		"tables":         []string{"g", "h"},
		"auto_generated": true,
		"nullable":       true,
	}

	var intf any = []any{
		map[string]any{
			"match":   fullColumn,
			"replace": columnWithImports(t, fullColumn, "abc", "github.com/abc"),
		},
	}

	typeReplace := ConvertReplacements(intf)
	if len(typeReplace) != 1 {
		t.Error("should have one entry")
	}

	checkColumn := func(t *testing.T, c drivers.Column) {
		t.Helper()
		if c.Name != "a" {
			t.Error("value was wrong:", c.Name)
		}
		if c.Type != "b" {
			t.Error("value was wrong:", c.Type)
		}
		if c.DBType != "c" {
			t.Error("value was wrong:", c.DBType)
		}
		if c.UDTName != "d" {
			t.Error("value was wrong:", c.UDTName)
		}
		if c.FullDBType != "e" {
			t.Error("value was wrong:", c.FullDBType)
		}
		if c.ArrType != "f" {
			t.Error("value was wrong:", c.ArrType)
		}
		if c.Generated != true {
			t.Error("value was wrong:", c.Generated)
		}
		if c.Nullable != true {
			t.Error("value was wrong:", c.Nullable)
		}
	}

	r := typeReplace[0]
	checkColumn(t, r.Match)
	checkColumn(t, r.Replace)

	if got := r.Replace.Imports[0]; got != "abc" {
		t.Error("standard import wrong:", got)
	}
	if got := r.Replace.Imports[1]; got != "github.com/abc" {
		t.Error("standard import wrong:", got)
	}
	if got := r.Tables; !reflect.DeepEqual(r.Tables, []string{"g", "h"}) {
		t.Error("tables in types.match wrong:", got)
	}
}
