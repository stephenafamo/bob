package driver

import (
	"fmt"
	"os"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
)

type colInfo struct {
	// Postgres only extension bits
	// ArrType is the underlying data type of the Postgres
	// ARRAY type. See here:
	// https://www.postgresql.org/docs/9.1/static/infoschema-element-types.html
	ArrType   string `json:"arr_type" yaml:"arr_type" toml:"arr_type"`
	UDTName   string `json:"udt_name" yaml:"udt_name" toml:"udt_name"`
	UDTSchema string `json:"udt_schema" yaml:"udt_schema" toml:"udt_schema"`
	// DomainName is the domain type name associated to the column. See here:
	// https://www.postgresql.org/docs/10/extend-type-system.html#EXTEND-TYPE-SYSTEM-DOMAINS
	DomainName string `json:"domain_name" toml:"domain_name"`
}

// translateColumnType converts postgres database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
func (d *Driver) translateColumnType(c drivers.Column, info colInfo) drivers.Column {
	typMap := d.typMap()

	switch c.DBType {
	case "bigint", "bigserial":
		c.Type = "int64"
	case "integer", "serial":
		c.Type = "int"
	case "oid":
		c.Type = "uint32"
	case "smallint", "smallserial":
		c.Type = "int16"
	case "decimal", "numeric":
		c.Type = "decimal.Decimal"
	case "double precision":
		c.Type = "float64"
	case "real":
		c.Type = "float32"
	case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "cidr", "inet", "macaddr", "text", "xml":
		c.Type = "string"
	case "json", "jsonb":
		c.Type = "types.JSON[json.RawMessage]"
	case "bytea":
		c.Type = "[]byte"
	case "boolean":
		c.Type = "bool"
	case "date", "time", "timestamp without time zone", "timestamp with time zone", "time without time zone", "time with time zone":
		c.Type = "time.Time"
	case "point":
		c.Type = "pgeo.Point"
	case "line":
		c.Type = "pgeo.Line"
	case "lseg":
		c.Type = "pgeo.Lseg"
	case "box":
		c.Type = "pgeo.Box"
	case "path":
		c.Type = "pgeo.Path"
	case "polygon":
		c.Type = "pgeo.Polygon"
	case "circle":
		c.Type = "pgeo.Circle"
	case "uuid":
		c.Type = "uuid.UUID"
	case "ENUM":
		c.Type = "string"
		for _, e := range d.enums {
			if e.Schema == info.UDTSchema && e.Name == info.UDTName {
				c.Type = e.Type
			}
		}
	case "ARRAY":
		var dbType string
		var imports importers.List
		c.Type, dbType, imports = d.getArrayType(info)
		c.Imports = append(c.Imports, imports...)
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

	c.Imports = append(c.Imports, typMap[c.Type]...)
	return c
}

// getArrayType returns the correct Array type for each database type
func (d *Driver) getArrayType(info colInfo) (string, string, importers.List) {
	typMap := d.typMap()

	if info.ArrType == "USER-DEFINED" {
		name := info.UDTName[1:] // postgres prefixes with an underscore
		for _, e := range d.enums {
			if e.Schema == info.UDTSchema && e.Name == name {
				return fmt.Sprintf("parray.EnumArray[%s]", e.Type), info.UDTName, typMap["parray"]
			}
		}
		return "pq.StringArray", info.ArrType, nil
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
			return "pq.Int64Array", info.ArrType, nil
		case "bytea":
			return "pq.ByteaArray", info.ArrType, nil
		case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "cidr", "inet", "macaddr", "text", "xml":
			return "pq.StringArray", info.ArrType, nil
		case "boolean":
			return "pq.BoolArray", info.ArrType, nil
		case "uuid":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["uuid.UUID"]...)
			return "parray.Array[uuid.UUID]", info.ArrType, imports
		case "decimal", "numeric":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["decimal.Decimal"]...)
			return "parray.Array[decimal.Decimal]", info.ArrType, imports
		case "double precision", "real":
			return "pq.Float64Array", info.ArrType, nil
		default:
			return "pq.StringArray", info.ArrType, nil
		}
	} else {
		switch info.UDTName {
		case "_int4", "_int8":
			return "pq.Int64Array", info.UDTName, nil
		case "_bytea":
			return "pq.ByteaArray", info.UDTName, nil
		case "_bit", "_interval", "_varbit", "_char", "_money", "_varchar", "_cidr", "_inet", "_macaddr", "_citext", "_text", "_xml":
			return "pq.StringArray", info.UDTName, nil
		case "_bool":
			return "pq.BoolArray", info.UDTName, nil
		case "_uuid":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["uuid.UUID"]...)
			return "parray.Array[uuid.UUID]", info.ArrType, imports
		case "_numeric":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["decimal.Decimal"]...)
			return "parray.Array[decimal.Decimal]", info.UDTName, imports
		case "_float4", "_float8":
			return "pq.Float64Array", info.UDTName, nil
		default:
			return "pq.StringArray", info.UDTName, nil
		}
	}
}

func (d *Driver) typMap() map[string]importers.List {
	var uuidPkg string
	switch d.config.UUIDPkg {
	case "google":
		uuidPkg = `"github.com/google/uuid"`
	default:
		uuidPkg = `"github.com/gofrs/uuid"`
	}

	return map[string]importers.List{
		"time.Time":                   {`"time"`},
		"pq.BoolArray":                {`"github.com/lib/pq"`},
		"pq.Int64Array":               {`"github.com/lib/pq"`},
		"pq.ByteaArray":               {`"github.com/lib/pq"`},
		"pq.StringArray":              {`"github.com/lib/pq"`},
		"pq.Float64Array":             {`"github.com/lib/pq"`},
		"uuid.UUID":                   {uuidPkg},
		"pgeo.Box":                    {`"github.com/saulortega/pgeo"`},
		"pgeo.Line":                   {`"github.com/saulortega/pgeo"`},
		"pgeo.Lseg":                   {`"github.com/saulortega/pgeo"`},
		"pgeo.Path":                   {`"github.com/saulortega/pgeo"`},
		"pgeo.Point":                  {`"github.com/saulortega/pgeo"`},
		"pgeo.Polygon":                {`"github.com/saulortega/pgeo"`},
		"decimal.Decimal":             {`"github.com/shopspring/decimal"`},
		"types.HStore":                {`"github.com/stephenafamo/bob/types"`},
		"parray":                      {`"github.com/stephenafamo/bob/types/parray"`},
		"types.JSON[json.RawMessage]": {`"encoding/json"`, `"github.com/stephenafamo/bob/types"`},
	}
}
