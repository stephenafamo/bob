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
