package parser

import (
	"testing"

	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
)

func TestTranslateArrayType(t *testing.T) {
	t.Parallel()

	enums := []Enum{{Schema: "public", Name: "status", Type: "Status"}}

	cases := []struct {
		name   string
		driver string
		info   ColInfo
		want   string
	}{
		{"pq text", "github.com/lib/pq", ColInfo{ArrType: "text"}, "pq.StringArray"},
		{"pq integer", "github.com/lib/pq", ColInfo{ArrType: "integer"}, "pq.Int32Array"},
		{"pq bytea", "github.com/lib/pq", ColInfo{ArrType: "bytea"}, "pq.ByteaArray"},
		{"pq numeric", "github.com/lib/pq", ColInfo{ArrType: "numeric"}, "pgtypes.Array[decimal.Decimal]"},
		{"pq unknown udt", "github.com/lib/pq", ColInfo{ArrType: "USER-DEFINED", UDTName: "_citext"}, "pq.StringArray"},
		{"pq enum", "github.com/lib/pq", ColInfo{ArrType: "USER-DEFINED", UDTSchema: "public", UDTName: "_status"}, "pgtypes.EnumArray[enums.Status]"},
		{"stdlib text", "github.com/jackc/pgx/v5/stdlib", ColInfo{ArrType: "text"}, "pq.StringArray"},
		{"pgx text", DriverPgx, ColInfo{ArrType: "text"}, "pgtypes.Array[string]"},
		{"pgx integer", DriverPgx, ColInfo{ArrType: "integer"}, "pgtypes.Array[int32]"},
		{"pgx bytea", DriverPgx, ColInfo{ArrType: "bytea"}, "pgtypes.Array[[]byte]"},
		{"pgx numeric", DriverPgx, ColInfo{ArrType: "numeric"}, "pgtypes.Array[decimal.Decimal]"},
		{"pgx unknown udt", DriverPgx, ColInfo{ArrType: "USER-DEFINED", UDTName: "_citext"}, "pgtypes.Array[string]"},
		{"pgx enum", DriverPgx, ColInfo{ArrType: "USER-DEFINED", UDTSchema: "public", UDTName: "_status"}, "pgtypes.EnumArray[enums.Status]"},
		{"pgx domain over array", DriverPgx, ColInfo{UDTName: "_int4"}, "pgtypes.Array[int32]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := &Translator{Types: helpers.Types(), Enums: enums, Driver: tc.driver}
			col := tr.TranslateColumnType(drivers.Column{DBType: "ARRAY"}, tc.info)
			if col.Type != tc.want {
				t.Fatalf("got %q, want %q", col.Type, tc.want)
			}

			typ := tr.Types.Index(col.Type)
			if typ.RandomExpr == "" {
				t.Fatalf("%s has no random expression", col.Type)
			}
		})
	}
}

func TestConfigurePlainSliceArrayTypes(t *testing.T) {
	t.Parallel()

	userNullType := drivers.NullType{Name: "EventsOrNull", ValidExpr: "SRC.Valid"}

	types := helpers.Types()
	types.SetTypeModifier(drivers.AarondlNull{})
	types.Register("[]types.JSON[Event]", drivers.Type{RandomExpr: "return nil"})
	types.Register("[]Custom", drivers.Type{NullType: userNullType})
	types.Register("Strings", drivers.Type{AliasOf: "[]string", RandomExpr: "return Strings{}"})
	types.Register("[]string", drivers.Type{RandomExpr: "return nil"})

	tables := drivers.Tables[any, any]{{
		Columns: []drivers.Column{
			{Name: "events", DBType: "jsonb[]", Type: "[]types.JSON[Event]", Nullable: true},
			{Name: "nnull_ints", DBType: "integer[]", Type: "[]int64"},
			{Name: "unregistered", DBType: "text[]", Type: "[]Unregistered", Nullable: true},
			{Name: "custom", DBType: "text[]", Type: "[]Custom", Nullable: true},
			{Name: "alias", DBType: "text[]", Type: "Strings", Nullable: true},
			{Name: "generated", DBType: "uuid[]", Type: "pgtypes.Array[uuid.UUID]", Nullable: true},
			{Name: "bytes", DBType: "bytea", Type: "[]byte", Nullable: true},
		},
	}}

	ConfigurePlainSliceArrayTypes(&types, tables)

	for _, typ := range []string{"[]types.JSON[Event]", "[]Unregistered", "[]string"} {
		nullType := types.GetNullType("", typ)
		if nullType.Name != typ {
			t.Errorf("%s: got null type %q, want the bare slice", typ, nullType.Name)
		}
		if got := types.GetNullTypeValid("", typ, "x"); got != "x != nil" {
			t.Errorf("%s: got valid expr %q, want %q", typ, got, "x != nil")
		}
	}

	if got := types.Index("[]types.JSON[Event]").RandomExpr; got != "return nil" {
		t.Errorf("random expression was not preserved: %q", got)
	}
	if got := types.Index("[]string").RandomExpr; got != "return nil" {
		t.Errorf("alias random expression leaked into its target: %q", got)
	}
	if got := types.Index("[]Custom").NullType; got.Name != userNullType.Name || got.ValidExpr != userNullType.ValidExpr {
		t.Errorf("user null type was overwritten: %+v", got)
	}

	for _, typ := range []string{"[]int64", "pgtypes.Array[uuid.UUID]", "[]byte"} {
		if got := types.Index(typ).NullType.Name; got != "" {
			t.Errorf("%s: unexpected null type %q", typ, got)
		}
	}
}

func TestTranslateTSVectorType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		driver string
		want   string
	}{
		{"github.com/lib/pq", "string"},
		{"github.com/jackc/pgx/v5/stdlib", "string"},
		{DriverPgx, "pgtypes.TSVector"},
	}

	for _, tc := range cases {
		t.Run(tc.driver, func(t *testing.T) {
			t.Parallel()

			tr := &Translator{Types: helpers.Types(), Driver: tc.driver}
			col := tr.TranslateColumnType(drivers.Column{DBType: "tsvector"}, ColInfo{})
			if col.Type != tc.want {
				t.Fatalf("got %q, want %q", col.Type, tc.want)
			}

			typ := tr.Types.Index(col.Type)
			if typ.RandomExpr == "" {
				t.Fatalf("%s has no random expression", col.Type)
			}
		})
	}
}
