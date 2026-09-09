package parser

import (
	"strings"
	"testing"

	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
)

func TestTranslateArrayTypesNativePgx(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		driver   string
		arrType  string
		wantType string
	}{
		{
			name:     "pq text array stays pq.StringArray",
			driver:   "github.com/lib/pq",
			arrType:  "text",
			wantType: "pq.StringArray",
		},
		{
			name:     "native pgx text array uses []string",
			driver:   DriverPgx,
			arrType:  "text",
			wantType: "[]string",
		},
		{
			name:     "native pgx int array uses []int32",
			driver:   DriverPgx,
			arrType:  "integer",
			wantType: "[]int32",
		},
		{
			name:     "native pgx jsonb array uses bare slice",
			driver:   DriverPgx,
			arrType:  "jsonb",
			wantType: "[]types.JSON[json.RawMessage]",
		},
		{
			name:     "native pgx alias array uses canonical Go type",
			driver:   DriverPgx,
			arrType:  "xml",
			wantType: "[]string",
		},
		{
			name:     "pq jsonb array uses pgtypes.Array",
			driver:   "github.com/lib/pq",
			arrType:  "jsonb",
			wantType: "pgtypes.Array[types.JSON[json.RawMessage]]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			types := helpers.Types()
			tr := &Translator{Types: types, Driver: tc.driver}
			col := tr.TranslateColumnType(drivers.Column{DBType: "ARRAY"}, ColInfo{ArrType: tc.arrType})
			if col.Type != tc.wantType {
				t.Fatalf("Type = %q, want %q", col.Type, tc.wantType)
			}
		})
	}
}

func TestTranslateArrayTypesNativePgxNullTypeWithModifier(t *testing.T) {
	t.Parallel()

	types := helpers.Types()
	types.SetTypeModifier(drivers.AarondlNull{})

	trPQ := &Translator{Types: types, Driver: "github.com/lib/pq"}
	pqCol := trPQ.TranslateColumnType(drivers.Column{DBType: "ARRAY"}, ColInfo{ArrType: "text"})
	pqNull := types.GetNullType("models", pqCol.Type)
	if pqNull.Name != "null.Val[pq.StringArray]" {
		t.Fatalf("pq NullType.Name = %q, want null.Val[pq.StringArray]", pqNull.Name)
	}

	typesPgx := helpers.Types()
	typesPgx.SetTypeModifier(drivers.AarondlNull{})
	trPgx := &Translator{Types: typesPgx, Driver: DriverPgx}
	pgxCol := trPgx.TranslateColumnType(drivers.Column{DBType: "ARRAY"}, ColInfo{ArrType: "text"})
	pgxNull := typesPgx.GetNullType("models", pgxCol.Type)
	if pgxNull.Name != "[]string" {
		t.Fatalf("pgx NullType.Name = %q, want []string", pgxNull.Name)
	}
	if got := typesPgx.GetNullTypeValid("models", pgxCol.Type, "v"); got != "v != nil" {
		t.Fatalf("GetNullTypeValid = %q, want %q", got, "v != nil")
	}
	wantCreate := "func() []string { if valid { return value }; return nil }()"
	gotCreate := typesPgx.GetNullType("models", pgxCol.Type).CreateExpr
	gotCreate = strings.NewReplacer(
		"NULLTYPE", "[]string",
		"NULLVAL", "valid",
		"SRC", "value",
	).Replace(gotCreate)
	if gotCreate != wantCreate {
		t.Fatalf("CreateExpr = %q, want %q", gotCreate, wantCreate)
	}
}

func TestTranslateEnumArrayTypesNativePgx(t *testing.T) {
	t.Parallel()

	types := helpers.Types()
	types.SetTypeModifier(drivers.AarondlNull{})
	tr := &Translator{
		Types:  types,
		Driver: DriverPgx,
		Enums: []Enum{{
			Schema: "public",
			Name:   "unicode_enum",
			Type:   "UnicodeEnum",
		}},
	}
	col := tr.TranslateColumnType(drivers.Column{DBType: "ARRAY"}, ColInfo{
		ArrType:   "USER-DEFINED",
		UDTName:   "_unicode_enum",
		UDTSchema: "public",
	})
	want := "pgtypes.EnumArray[enums.UnicodeEnum]"
	if col.Type != want {
		t.Fatalf("Type = %q, want %q", col.Type, want)
	}
	nullTyp := types.GetNullType("models", col.Type)
	wantNull := "null.Val[" + want + "]"
	if nullTyp.Name != wantNull {
		t.Fatalf("NullType.Name = %q, want %q", nullTyp.Name, wantNull)
	}
	if got := types.GetNullTypeValid("models", col.Type, "v"); got != "v.IsValue()" {
		t.Fatalf("GetNullTypeValid = %q, want %q", got, "v.IsValue()")
	}
	if types.CanCompareWithEquals("models", col.Type) {
		t.Fatal("enum array must not be treated as ==-comparable")
	}
}

func TestConfigureNativePgxArrayTypesAfterOverride(t *testing.T) {
	t.Parallel()

	const customArray = "[]types.JSON[Event]"
	const explicitlyConfiguredArray = "[]Custom"
	types := helpers.Types()
	types.Register(customArray, drivers.Type{})
	types.Register(explicitlyConfiguredArray, drivers.Type{
		NullType: drivers.NullType{Name: "custom.NullArray"},
	})
	tables := drivers.Tables[any, any]{
		{
			Columns: []drivers.Column{
				{
					DBType: "jsonb[]",
					Type:   customArray,
				},
				{
					DBType: "text[]",
					Type:   explicitlyConfiguredArray,
				},
			},
		},
	}

	ConfigureNativePgxArrayTypes(types, tables)

	nullTyp := types.GetNullType("models", customArray)
	if nullTyp.Name != customArray {
		t.Fatalf("NullType.Name = %q, want %q", nullTyp.Name, customArray)
	}
	if got := types.GetNullTypeValid("models", customArray, "v"); got != "v != nil" {
		t.Fatalf("GetNullTypeValid = %q, want %q", got, "v != nil")
	}
	if got := types.GetNullType("models", explicitlyConfiguredArray).Name; got != "custom.NullArray" {
		t.Fatalf("explicit NullType.Name = %q, want %q", got, "custom.NullArray")
	}
}
