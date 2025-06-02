package parser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
)

const pgtypesImport = `"github.com/stephenafamo/bob/types/pgtypes"`

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
	mu    sync.Mutex
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
	case "boolean":
		c.Type = "bool"
	case "date", "time", "timestamp without time zone", "timestamp with time zone", "time without time zone", "time with time zone":
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
	default:
		switch info.UDTName {
		case "hstore":
			c.Type = "pgtypes.HStore"
			c.DBType = "hstore"
		case "citext":
			c.Type = "string"
		default:
			c.Type = "string"
		}
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
		return "pq.StringArray", name
	}

	typToTranslate := info.ArrType

	if typToTranslate == "" {
		typToTranslate = info.UDTName[1:] // postgres prefixes with an underscore
	}

	translated := t.TranslateColumnType(
		drivers.Column{DBType: typToTranslate}, ColInfo{},
	).Type

	switch translated {
	case "bool":
		return "pq.BoolArray", typToTranslate
	case "int32":
		return "pq.Int32Array", typToTranslate
	case "int64":
		return "pq.Int64Array", typToTranslate
	case "float32":
		return "pq.Float32Array", typToTranslate
	case "float64":
		return "pq.Float64Array", typToTranslate
	case "string":
		return "pq.StringArray", typToTranslate
	case "[]byte":
		return "pq.ByteaArray", typToTranslate
	default:
		return t.addPgGenericArrayType(t.Types, translated), typToTranslate
	}
}

func (t *Translator) addPgEnumArrayType(types drivers.Types, enumTyp string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	arrTyp := fmt.Sprintf("pgtypes.EnumArray[%s]", enumTyp)

	// premptively add the enum type
	// this is to prevent issues if the enum is only used in an array
	helpers.EnumType(types, enumTyp)

	types.Register(arrTyp, drivers.Type{
		DependsOn:           []string{enumTyp},
		Imports:             []string{pgtypesImport},
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f, limits...)
            }
            return arr`, arrTyp, gen.NormalizeType(enumTyp)),
	})

	return arrTyp
}

func (d *Translator) addPgGenericArrayType(types drivers.Types, singleTyp string) string {
	d.mu.Lock()
	defer d.mu.Unlock()

	singleTypDef := types.Index(singleTyp)
	singleComparer := strings.ReplaceAll(singleTypDef.CompareExpr, "AAA", "a")
	singleComparer = strings.ReplaceAll(singleComparer, "BBB", "b")
	if singleComparer == "" {
		singleComparer = "a == b"
	}

	typ := fmt.Sprintf("pgtypes.Array[%s]", singleTyp)

	types.Register(typ, drivers.Type{
		DependsOn: []string{singleTyp},
		Imports:   append([]string{pgtypesImport}, singleTypDef.Imports...),
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f, limits...)
            }
            return arr`, typ, gen.NormalizeType(singleTyp)),
		CompareExpr: fmt.Sprintf(`slices.EqualFunc(AAA, BBB, func(a, b %s) bool {
                return %s
            })`, singleTyp, singleComparer),
		CompareExprImports: append(append(
			[]string{`"slices"`},
			singleTypDef.CompareExprImports...),
			singleTypDef.Imports...),
	})

	return typ
}
