package parser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
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
	case "decimal", "numeric", "money":
		c.Type = "decimal.Decimal"
	case "double precision":
		c.Type = "float64"
	case "real":
		c.Type = "float32"
	case "bit", "interval", "uuint", "bit varying", "character", "character varying", "text":
		c.Type = "string"
	case "xml":
		c.Type = "xml"
	case "json", "jsonb":
		c.Type = "types.JSON[json.RawMessage]"
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
		c.Type = "types.Text[netip.Addr, *netip.Addr]"
	case "macaddr", "macaddr8":
		c.Type = "pgtypes.Macaddr"
	case "pg_lsn":
		c.Type = "pgtypes.LSN"
	case "txid_snapshot":
		c.Type = "pgtypes.TxIDSnapshot"
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

	// If a domain is created with a statement like this: "CREATE DOMAIN
	// text_array AS TEXT[] CHECK ( ... )" then the array type will be null,
	// but the udt name will be whatever the underlying type is with a leading
	// underscore. Note that this code handles some types, but not nearly all
	// the possibities. Notably, an array of a user-defined type ("CREATE
	// DOMAIN my_array AS my_type[]") will be treated as an array of strings,
	// which is not guaranteed to be correct.
	if info.ArrType != "" {
		switch info.ArrType {
		case "bigint", "bigserial", "integer", "serial", "smallint", "smallserial", "oid":
			return "pq.Int64Array", info.ArrType
		case "bytea":
			return "pq.ByteaArray", info.ArrType
		case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "cidr", "inet", "macaddr", "text", "xml":
			return "pq.StringArray", info.ArrType
		case "boolean":
			return "pq.BoolArray", info.ArrType
		case "uuid":
			typ := t.addPgGenericArrayType(t.Types, "uuid.UUID")
			return typ, info.ArrType
		case "decimal", "numeric":
			typ := t.addPgGenericArrayType(t.Types, "decimal.Decimal")
			return typ, info.ArrType
		case "double precision", "real":
			return "pq.Float64Array", info.ArrType
		default:
			return "pq.StringArray", info.ArrType
		}
	} else {
		switch info.UDTName {
		case "_int4", "_int8":
			return "pq.Int64Array", info.UDTName
		case "_bytea":
			return "pq.ByteaArray", info.UDTName
		case "_bit", "_interval", "_varbit", "_char", "_money", "_varchar", "_cidr", "_inet", "_macaddr", "_citext", "_text", "_xml":
			return "pq.StringArray", info.UDTName
		case "_bool":
			return "pq.BoolArray", info.UDTName
		case "_uuid":
			typ := t.addPgGenericArrayType(t.Types, "uuid.UUID")
			return typ, info.UDTName
		case "_numeric":
			typ := t.addPgGenericArrayType(t.Types, "decimal.Decimal")
			return typ, info.UDTName
		case "_float4", "_float8":
			return "pq.Float64Array", info.UDTName
		default:
			return "pq.StringArray", info.UDTName
		}
	}
}

func (t *Translator) addPgEnumArrayType(types drivers.Types, enumTyp string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	arrTyp := fmt.Sprintf("pgtypes.EnumArray[%s]", enumTyp)

	// premptively add the enum type
	// this is to prevent issues if the enum is only used in an array
	helpers.EnumType(types, enumTyp)

	types[arrTyp] = drivers.Type{
		DependsOn:           []string{enumTyp},
		Imports:             language.ImportList{pgtypesImport},
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f)
            }
            return arr`, arrTyp, gen.NormalizeType(enumTyp)),
	}

	return arrTyp
}

func (d *Translator) addPgGenericArrayType(types drivers.Types, singleTyp string) string {
	d.mu.Lock()
	defer d.mu.Unlock()

	singleTypDef := types[singleTyp]
	singleComparer := strings.ReplaceAll(singleTypDef.CompareExpr, "AAA", "a")
	singleComparer = strings.ReplaceAll(singleComparer, "BBB", "b")
	if singleComparer == "" {
		singleComparer = "a == b"
	}

	typ := fmt.Sprintf("pgtypes.Array[%s]", singleTyp)

	types[typ] = drivers.Type{
		DependsOn: []string{singleTyp},
		Imports:   append(language.ImportList{pgtypesImport}, singleTypDef.Imports...),
		RandomExpr: fmt.Sprintf(`arr := make(%s, f.IntBetween(1, 5))
            for i := range arr {
                arr[i] = random_%s(f)
            }
            return arr`, typ, gen.NormalizeType(singleTyp)),
		CompareExpr: fmt.Sprintf(`slices.EqualFunc(AAA, BBB, func(a, b %s) bool {
                return %s
            })`, singleTyp, singleComparer),
		CompareExprImports: append(append(
			language.ImportList{`"slices"`},
			singleTypDef.CompareExprImports...),
			singleTypDef.Imports...),
	}

	return typ
}
