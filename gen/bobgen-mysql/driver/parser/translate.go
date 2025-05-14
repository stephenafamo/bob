package parser

import (
	"regexp"
	"strings"
)

var rgxHasBrackets = regexp.MustCompile(`^(\w+)(\((\d+)(,(\d+))?\))?`)

//nolint:nonamedreturns
func getParts(fullType string) (name string, l1 string, l2 string, unsigned bool) {
	unsigned = strings.HasSuffix(fullType, " unsigned")

	parts := rgxHasBrackets.FindStringSubmatch(fullType)
	if len(parts) == 0 {
		return fullType, "", "", unsigned
	}

	return parts[1], parts[3], parts[5], unsigned
}

// translateTableColumnType converts mysql database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
func TranslateColumnType(fullType string) string {
	dbType, limit1, _, unsigned := getParts(fullType)
	switch dbType {
	case "tinyint":
		if unsigned {
			return "uint8"
		}

		if limit1 == "1" {
			// TINYINT(1) is a special case in MySQL, it is treated as a boolean
			return "bool"
		}

		return "int8"

	case "smallint":
		if unsigned {
			return "uint16"
		}
		return "int16"

	case "mediumint":
		if unsigned {
			return "uint32"
		}
		return "int32"

	case "int", "integer":
		if unsigned {
			return "uint32"
		}
		return "int32"

	case "bigint":
		if unsigned {
			return "uint64"
		}
		return "int64"

	case "float":
		return "float32"

	case "double", "double precision", "real":
		return "float64"

	case "boolean", "bool":
		return "bool"

	case "date", "datetime", "timestamp":
		return "time.Time"

	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		return "[]byte"

	case "numeric", "decimal", "dec", "fixed":
		return "decimal.Decimal"

	case "json":
		return "types.JSON[json.RawMessage]"

	default:
		return "string"
	}
}
