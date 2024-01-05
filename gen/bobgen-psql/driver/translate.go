package driver

import (
	"fmt"
	"os"

	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
)

type colInfo struct {
	// Postgres only extension bits
	// ArrType is the underlying data type of the Postgres
	// ARRAY type. See here:
	// https://www.postgresql.org/docs/9.1/static/infoschema-element-types.html
	ArrType   string `json:"arr_type" yaml:"arr_type" toml:"arr_type"`
	UDTName   string `json:"udt_name" yaml:"udt_name" toml:"udt_name"`
	UDTSchema string `json:"udt_schema" yaml:"udt_schema" toml:"udt_schema"`
}

// translateColumnType converts postgres database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
//
//nolint:gocyclo
func (d *driver) translateColumnType(c drivers.Column, info colInfo) drivers.Column {
	switch c.DBType {
	case "bigint":
		c.Type = "int64"
	case "bigserial":
		c.Type = "uint64"
	case "integer":
		c.Type = "int32"
	case "serial":
		c.Type = "uint32"
	case "oid":
		c.Type = "uint32"
	case "smallint":
		c.Type = "int16"
	case "smallserial":
		c.Type = "uint16"
	case "decimal", "numeric":
		c.Type = "decimal.Decimal"
	case "double precision":
		c.Type = "float64"
	case "real":
		c.Type = "float32"
	case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "text", "xml":
		c.Type = "string"
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
	case "inet", "cidr":
		c.Type = "netip.Addr"
	case "macaddr":
		c.Type = "net.HardwareAddr"
	case "ENUM":
		c.Type = "string"
		for _, e := range d.enums {
			if e.Schema == info.UDTSchema && e.Name == info.UDTName {
				c.Type = helpers.AddPgEnumType(d.types, e.Type)
			}
		}
	case "ARRAY":
		var dbType string
		c.Type, dbType = d.getArrayType(info)
		// Make DBType something like ARRAYinteger for parsing with randomize.Struct
		c.DBType = dbType + "[]"
	case "USER-DEFINED":
		switch info.UDTName {
		case "hstore":
			c.Type = "types.HStore"
			c.DBType = "hstore"
		case "citext":
			c.Type = "string"
		default:
			c.Type = "string"
			fmt.Fprintf(os.Stderr, "warning: incompatible data type detected: %s\n", info.UDTName)
		}
	default:
		c.Type = "string"
	}

	return c
}

// getArrayType returns the correct Array type for each database type
func (d *driver) getArrayType(info colInfo) (string, string) {
	if info.ArrType == "USER-DEFINED" {
		name := info.UDTName[1:] // postgres prefixes with an underscore
		for _, e := range d.enums {
			if e.Schema == info.UDTSchema && e.Name == name {
				typ := helpers.AddPgEnumArrayType(d.types, e.Type)
				return typ, info.UDTName
			}
		}
		return "pq.StringArray", info.ArrType
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
			typ := helpers.AddPgGenericArrayType(d.types, "uuid.UUID")
			return typ, info.ArrType
		case "decimal", "numeric":
			typ := helpers.AddPgGenericArrayType(d.types, "decimal.Decimal")
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
			typ := helpers.AddPgGenericArrayType(d.types, "uuid.UUID")
			return typ, info.UDTName
		case "_numeric":
			typ := helpers.AddPgGenericArrayType(d.types, "decimal.Decimal")
			return typ, info.UDTName
		case "_float4", "_float8":
			return "pq.Float64Array", info.UDTName
		default:
			return "pq.StringArray", info.UDTName
		}
	}
}
