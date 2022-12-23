package driver

import (
	"fmt"
	"os"
	"strings"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
	"github.com/volatiletech/strmangle"
)

// translateColumnType converts postgres database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
func (p *Driver) translateColumnType(c drivers.Column) drivers.Column {
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
	case "ARRAY":
		var dbType string
		if _, ok := p.enums[c.UDTName[1:]]; ok {
			enumName := c.UDTName[1:]
			dbType = fmt.Sprintf("enum.%s", enumName)
			c.Type = fmt.Sprintf("parray.EnumArray[%s]", strmangle.TitleCase(enumName))
			c.Imports = append(c.Imports, typMap["parray"]...)
		} else {
			var imports importers.List
			c.Type, dbType, imports = getArrayType(c)
			c.Imports = append(c.Imports, imports...)
		}
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
		if strings.HasPrefix(c.DBType, "enum.") {
			c.Type = strmangle.TitleCase(strings.TrimPrefix(c.DBType, "enum."))
		} else {
			c.Type = "string"
		}
	}

	c.Imports = append(c.Imports, typMap[c.Type]...)
	return c
}

// getArrayType returns the correct Array type for each database type
func getArrayType(c drivers.Column) (string, string, importers.List) {
	// If a domain is created with a statement like this: "CREATE DOMAIN
	// text_array AS TEXT[] CHECK ( ... )" then the array type will be null,
	// but the udt name will be whatever the underlying type is with a leading
	// underscore. Note that this code handles some types, but not nearly all
	// the possibities. Notably, an array of a user-defined type ("CREATE
	// DOMAIN my_array AS my_type[]") will be treated as an array of strings,
	// which is not guaranteed to be correct.
	if c.ArrType != nil {
		switch *c.ArrType {
		case "bigint", "bigserial", "integer", "serial", "smallint", "smallserial", "oid":
			return "pq.Int64Array", *c.ArrType, nil
		case "bytea":
			return "pq.ByteaArray", *c.ArrType, nil
		case "bit", "interval", "uuint", "bit varying", "character", "money", "character varying", "cidr", "inet", "macaddr", "text", "uuid", "xml":
			return "pq.StringArray", *c.ArrType, nil
		case "boolean":
			return "pq.BoolArray", *c.ArrType, nil
		case "decimal", "numeric":
			var imports importers.List
			imports = append(imports, typMap["parray"]...)
			imports = append(imports, typMap["decimal.Decimal"]...)
			return "parray.GenericArray[decimal.Decimal]", *c.ArrType, imports
		case "double precision", "real":
			return "pq.Float64Array", *c.ArrType, nil
		default:
			return "pq.StringArray", *c.ArrType, nil
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
