package driver

import (
	"fmt"
	"os"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
)

// translateColumnType converts postgres database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
func (d *Driver) translateColumnType(c drivers.Column) drivers.Column {
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
	case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "cidr", "inet", "macaddr", "text", "uuid", "xml":
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
	case "ENUM":
		c.Type = "string"
		for _, e := range d.enums {
			if e.Schema == c.UDTSchema && e.Name == c.UDTName {
				c.Type = e.Type
			}
		}
	case "ARRAY":
		var dbType string
		var imports importers.List
		c.Type, dbType, imports = d.getArrayType(c)
		c.Imports = append(c.Imports, imports...)
		// Make DBType something like ARRAYinteger for parsing with randomize.Struct
		c.DBType += dbType
	case "USER-DEFINED":
		switch c.UDTName {
		case "hstore":
			c.Type = "types.HStore"
			c.DBType = "hstore"
		case "citext":
			c.Type = "string"
		default:
			c.Type = "string"
			fmt.Fprintf(os.Stderr, "warning: incompatible data type detected: %s\n", c.UDTName)
		}
	default:
		c.Type = "string"
	}

	c.Imports = append(c.Imports, typMap[c.Type]...)
	return c
}

// getArrayType returns the correct Array type for each database type
func (d *Driver) getArrayType(c drivers.Column) (string, string, importers.List) {
	if c.ArrType == "USER-DEFINED" {
		name := c.UDTName[1:] // postgres prefixes with an underscore
		for _, e := range d.enums {
			if e.Schema == c.UDTSchema && e.Name == name {
				return fmt.Sprintf("parray.EnumArray[%s]", e.Type), c.UDTName, typMap["parray"]
			}
		}
		return "pq.StringArray", c.ArrType, nil
	}

	// If a domain is created with a statement like this: "CREATE DOMAIN
	// text_array AS TEXT[] CHECK ( ... )" then the array type will be null,
	// but the udt name will be whatever the underlying type is with a leading
	// underscore. Note that this code handles some types, but not nearly all
	// the possibities. Notably, an array of a user-defined type ("CREATE
	// DOMAIN my_array AS my_type[]") will be treated as an array of strings,
	// which is not guaranteed to be correct.
	if c.ArrType != "" {
		switch c.ArrType {
		case "bigint", "bigserial", "integer", "serial", "smallint", "smallserial", "oid":
			return "pq.Int64Array", c.ArrType, nil
		case "bytea":
			return "pq.ByteaArray", c.ArrType, nil
		case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "cidr", "inet", "macaddr", "text", "uuid", "xml":
			return "pq.StringArray", c.ArrType, nil
		case "boolean":
			return "pq.BoolArray", c.ArrType, nil
		case "decimal", "numeric":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["decimal.Decimal"]...)
			return "parray.GenericArray[decimal.Decimal]", c.ArrType, imports
		case "double precision", "real":
			return "pq.Float64Array", c.ArrType, nil
		default:
			return "pq.StringArray", c.ArrType, nil
		}
	} else {
		switch c.UDTName {
		case "_int4", "_int8":
			return "pq.Int64Array", c.UDTName, nil
		case "_bytea":
			return "pq.ByteaArray", c.UDTName, nil
		case "_bit", "_interval", "_varbit", "_char", "_money", "_varchar", "_cidr", "_inet", "_macaddr", "_citext", "_text", "_uuid", "_xml":
			return "pq.StringArray", c.UDTName, nil
		case "_bool":
			return "pq.BoolArray", c.UDTName, nil
		case "_numeric":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["decimal.Decimal"]...)
			return "parray.GenericArray[decimal.Decimal]", c.UDTName, imports
		case "_float4", "_float8":
			return "pq.Float64Array", c.UDTName, nil
		default:
			return "pq.StringArray", c.UDTName, nil
		}
	}
}

//nolint:gochecknoglobals
var typMap = map[string]importers.List{
	"time.Time":                   {`"time"`},
	"types.JSON[json.RawMessage]": {`"encoding/json"`, `"github.com/stephenafamo/bob/types"`},
	"decimal.Decimal":             {`"github.com/shopspring/decimal"`},
	"types.HStore":                {`"github.com/stephenafamo/bob/types"`},
	"pgeo.Point":                  {`"github.com/saulortega/pgeo"`},
	"pgeo.Line":                   {`"github.com/saulortega/pgeo"`},
	"pgeo.Lseg":                   {`"github.com/saulortega/pgeo"`},
	"pgeo.Box":                    {`"github.com/saulortega/pgeo"`},
	"pgeo.Path":                   {`"github.com/saulortega/pgeo"`},
	"pgeo.Polygon":                {`"github.com/saulortega/pgeo"`},
	"pq.ByteaArray":               {`"github.com/lib/pq"`},
	"pq.Int64Array":               {`"github.com/lib/pq"`},
	"pq.Float64Array":             {`"github.com/lib/pq"`},
	"pq.BoolArray":                {`"github.com/lib/pq"`},
	"pq.StringArray":              {`"github.com/lib/pq"`},
	"parray":                      {`"github.com/stephenafamo/bob/types/parray"`},
}
