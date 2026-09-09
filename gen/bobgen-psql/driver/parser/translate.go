package parser

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
)

const (
	pgtypesImport = `"github.com/stephenafamo/bob/types/pgtypes"`

	// DriverPgx is the module path for the native pgx/v5 interface (not stdlib).
	DriverPgx = "github.com/jackc/pgx/v5"
)

type Enum struct {
	Schema string
	Name   string
	Type   string
	Values pq.StringArray
}

type ColInfo struct {
	// Postgres only extension bits
	// ArrType is the underlying data type of the Postgres
	// ARRAY type. See here:
	// https://www.postgresql.org/docs/9.1/static/infoschema-element-types.html
	ArrType   string `json:"arr_type" yaml:"arr_type"`
	UDTName   string `json:"udt_name" yaml:"udt_name"`
	UDTSchema string `json:"udt_schema" yaml:"udt_schema"`
}

type Translator struct {
	Enums []Enum
	Types drivers.Types
	// Driver is the configured database driver module path (e.g. DriverPgx).
	// Native pgx scans arrays into []T directly; wrapping those in null.Val[[]T]
	// breaks scanning because pgx prefers sql.Scanner over its array plans.
	Driver string
	mu     sync.Mutex
}

//nolint:gocyclo
func (t *Translator) TranslateColumnType(c drivers.Column, info ColInfo) drivers.Column {
	switch c.DBType {
	case "bigint", "int8":
		c.Type = "int64"
	case "bigserial":
		c.Type = "uint64"
	case "integer", "int", "int4":
		c.Type = "int32"
	case "serial":
		c.Type = "uint32"
	case "oid":
		c.Type = "uint32"
	case "smallint", "int2":
		c.Type = "int16"
	case "smallserial":
		c.Type = "uint16"
	case "decimal", "numeric":
		c.Type = "decimal.Decimal"
	case "double precision":
		c.Type = "float64"
	case "real":
		c.Type = "float32"
	case "bit", "interval", "uuint", "bit varying", "character", "character varying", "text":
		c.Type = "string"
	case "xml":
		c.Type = "xml"
	case "money":
		c.Type = "money"
	case "json", "jsonb":
		c.Type = "types.JSON[json.RawMessage]"
	case "char", `"char"`:
		c.Type = "string" // should be a single character, but we treat it as a string
	case "bytea":
		c.Type = "[]byte"
	case "bool", "boolean":
		c.Type = "bool"
	case "date", "time",
		"timestamp", "timestamp without time zone",
		"timestamptz", "timestamp with time zone",
		"time with time zone", "time without time zone":
		c.Type = "time.Time"
	case "box":
		c.Type = "pgeo.Box"
	case "circle":
		c.Type = "pgeo.Circle"
	case "line":
		c.Type = "pgeo.Line"
	case "lseg":
		c.Type = "pgeo.Lseg"
	case "path":
		c.Type = "pgeo.Path"
	case "point":
		c.Type = "pgeo.Point"
	case "polygon":
		c.Type = "pgeo.Polygon"
	case "uuid":
		c.Type = "uuid.UUID"
	case "inet":
		c.Type = "pgtypes.Inet"
	case "cidr":
		c.Type = "types.Text[netip.Prefix, *netip.Prefix]"
	case "macaddr", "macaddr8":
		c.Type = "pgtypes.Macaddr"
	case "pg_lsn":
		c.Type = "pgtypes.LSN"
	case "txid_snapshot", "pg_snapshot":
		c.Type = "pgtypes.Snapshot"
	case "ENUM":
		c.Type = "string"
		c.DBType = info.UDTSchema + "." + info.UDTName
		for _, e := range t.Enums {
			if e.Schema == info.UDTSchema && e.Name == info.UDTName {
				t.mu.Lock()
				c.Type = helpers.EnumType(t.Types, e.Type)
				t.mu.Unlock()
			}
		}
	case "ARRAY":
		var dbType string
		c.Type, dbType = t.getArrayType(info)
		c.DBType = dbType + "[]"

	case "USER-DEFINED":
		c.DBType = info.UDTName
		switch info.UDTName {
		case "hstore":
			c.Type = "pgtypes.HStore"
		case "vector":
			c.Type = "pgvector.Vector"
		case "halfvec":
			c.Type = "pgvector.HalfVector"
		case "sparsevec":
			c.Type = "pgvector.SparseVector"
		default:
			c.Type = "string"
		}

	default:
		c.Type = "string"
	}

	return c
}

func (t *Translator) getArrayType(info ColInfo) (string, string) {
	if info.ArrType == "USER-DEFINED" {
		name := info.UDTName[1:] // postgres prefixes with an underscore
		for _, e := range t.Enums {
			if e.Schema == info.UDTSchema && e.Name == name {
				typ := t.addPgEnumArrayType(t.Types, e.Type)
				return typ, info.UDTName
			}
		}
		if t.Driver == DriverPgx {
			return t.addPgNativeSliceArrayType(t.Types, "string"), name
		}
		return "pq.StringArray", name
	}

	typToTranslate := info.ArrType

	if typToTranslate == "" {
		typToTranslate = info.UDTName[1:] // postgres prefixes with an underscore
	}

	translated := t.TranslateColumnType(
		drivers.Column{DBType: typToTranslate}, ColInfo{},
	).Type

	if t.Driver == DriverPgx {
		return t.addPgNativeSliceArrayType(t.Types, translated), typToTranslate
	}

	if pqTyp, ok := pqSliceArrayType(translated); ok {
		return pqTyp, typToTranslate
	}
	return t.addPgGenericArrayType(t.Types, translated), typToTranslate
}

func pqSliceArrayType(typ string) (string, bool) {
	switch typ {
	case "bool":
		return "pq.BoolArray", true
	case "int32":
		return "pq.Int32Array", true
	case "int64":
		return "pq.Int64Array", true
	case "float32":
		return "pq.Float32Array", true
	case "float64":
		return "pq.Float64Array", true
	case "string":
		return "pq.StringArray", true
	case "[]byte":
		return "pq.ByteaArray", true
	default:
		return "", false
	}
}

// nilSentinelNullType configures nullable columns to use the bare slice type.
// A nil slice represents SQL NULL; a non-nil (possibly empty) slice is a value.
// This avoids null.Val[[]T], which breaks native pgx array scanning.
func nilSentinelNullType(name string) drivers.NullType {
	return drivers.NullType{
		Name:       name,
		ValidExpr:  "SRC != nil",
		UseExpr:    "SRC",
		CreateExpr: "func() NULLTYPE { if NULLVAL { return SRC }; return nil }()",
	}
}

func arrayRandomExpr(arrTyp, elemTyp string) string {
	return fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f, limits...)
            }
            return arr`, arrTyp, gen.NormalizeType(elemTyp))
}

func elementComparer(typeDef drivers.Type) string {
	comparer := strings.ReplaceAll(typeDef.CompareExpr, "AAA", "a")
	comparer = strings.ReplaceAll(comparer, "BBB", "b")
	if comparer == "" {
		return "a == b"
	}
	return comparer
}

func sliceArrayCompareExpr(elemTyp string, elemDef drivers.Type) (string, []string) {
	if elemTyp == "[]byte" {
		return `slices.EqualFunc(AAA, BBB, func(a, b []byte) bool {
                return bytes.Equal(a, b)
            })`, []string{`"slices"`, `"bytes"`}
	}
	if elemDef.CompareExpr != "" {
		return fmt.Sprintf(`slices.EqualFunc(AAA, BBB, func(a, b %s) bool {
                return %s
            })`, elemTyp, elementComparer(elemDef)), append([]string{`"slices"`}, elemDef.CompareExprImports...)
	}
	return `slices.Equal(AAA, BBB)`, []string{`"slices"`}
}

func (t *Translator) addPgNativeSliceArrayType(types drivers.Types, elemTyp string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	goElemTyp, elemDef := types.GetNameAndDef("", elemTyp)
	typ := "[]" + goElemTyp
	if types.Contains(typ) {
		return typ
	}

	compareExpr, compareImports := sliceArrayCompareExpr(goElemTyp, elemDef)

	types.Register(typ, drivers.Type{
		DependsOn:           []string{elemTyp},
		Imports:             slices.Clone(elemDef.Imports),
		NoScannerValuerTest: true,
		NoRandomizationTest: elemDef.NoRandomizationTest,
		RandomExpr:          arrayRandomExpr(typ, elemTyp),
		CompareExpr:         compareExpr,
		CompareExprImports:  append(compareImports, elemDef.Imports...),
		NullType:            nilSentinelNullType(typ),
	})

	return typ
}

func (t *Translator) addPgEnumArrayType(types drivers.Types, enumTyp string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	// preemptively add the enum type
	// this is to prevent issues if the enum is only used in an array
	fullEnumTyp := helpers.EnumType(types, enumTyp)
	arrTyp := fmt.Sprintf("pgtypes.EnumArray[enums.%s]", enumTyp)

	typeDef := drivers.Type{
		DependsOn:           []string{fullEnumTyp},
		Imports:             []string{"output(enums)", pgtypesImport},
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr:          arrayRandomExpr(arrTyp, fullEnumTyp),
		CompareExpr:         "slices.Equal(AAA, BBB)",
		CompareExprImports:  []string{`"slices"`},
	}

	types.Register(arrTyp, typeDef)

	return arrTyp
}

func (t *Translator) addPgGenericArrayType(types drivers.Types, singleTyp string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	singleTypDef := types.Index(singleTyp)
	singleComparer := elementComparer(singleTypDef)
	typ := fmt.Sprintf("pgtypes.Array[%s]", singleTyp)

	types.Register(typ, drivers.Type{
		DependsOn:  []string{singleTyp},
		Imports:    append([]string{pgtypesImport}, singleTypDef.Imports...),
		RandomExpr: arrayRandomExpr(typ, singleTyp),
		CompareExpr: fmt.Sprintf(`slices.EqualFunc(AAA, BBB, func(a, b %s) bool {
                return %s
            })`, singleTyp, singleComparer),
		CompareExprImports: append(append(
			[]string{`"slices"`},
			singleTypDef.CompareExprImports...,
		),
			singleTypDef.Imports...),
	})

	return typ
}

// ConfigureNativePgxArrayTypes reapplies nil-sentinel nullability after
// user-defined types and replacements have been merged into the type registry.
func ConfigureNativePgxArrayTypes[C, I any](types drivers.Types, tables drivers.Tables[C, I]) {
	for _, table := range tables {
		for _, column := range table.Columns {
			if !strings.HasSuffix(strings.ToLower(column.DBType), "[]") {
				continue
			}
			if !strings.HasPrefix(column.Type, "[]") {
				continue
			}
			if !types.Contains(column.Type) {
				continue
			}

			typeDef := types.Index(column.Type)
			if typeDef.NullType.Name != "" {
				continue
			}

			typeDef.NullType = nilSentinelNullType(column.Type)
			types.Register(column.Type, typeDef)
		}
	}
}
