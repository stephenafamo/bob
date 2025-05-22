package parser

import (
	"regexp"
	"strings"
)

var rgxHasBrackets = regexp.MustCompile(`^(\w+)(\((\d+)(,(\d+))?\))?`)

//nolint:nonamedreturns
func getParts(fullType string) (name string, limits []string, unsigned bool) {
	unsigned = strings.HasSuffix(fullType, " unsigned")

	parts := rgxHasBrackets.FindStringSubmatch(fullType)
	if len(parts) == 0 {
		return fullType, nil, unsigned
	}

	limits = make([]string, 0, 2)
	if parts[3] != "" {
		limits = append(limits, parts[3])
	}
	if parts[5] != "" {
		limits = append(limits, parts[5])
	}

	return parts[1], limits, unsigned
}

// translateTableColumnType converts mysql database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
func TranslateColumnType(fullType string) (string, []string) {
	dbType, limits, unsigned := getParts(fullType)
	switch strings.ToLower(dbType) {
	case "tinyint":
		if unsigned {
			return "uint8", limits
		}

		if len(limits) > 0 && limits[0] == "1" {
			// TINYINT(1) is a special case in MySQL, it is treated as a boolean
			return "bool", limits
		}

		return "int8", limits

	case "smallint":
		if unsigned {
			return "uint16", limits
		}
		return "int16", limits

	case "mediumint": // 24-bit integer but no native Go type
		if unsigned {
			return "uint16", limits
		}
		return "int16", limits

	case "int", "integer":
		if unsigned {
			return "uint32", limits
		}
		return "int32", limits

	case "bigint":
		if unsigned {
			return "uint64", limits
		}
		return "int64", limits

	case "float":
		return "float32", limits

	case "double", "double precision", "real":
		return "float64", limits

	case "boolean", "bool":
		return "bool", limits

	case "date", "datetime", "timestamp":
		return "time.Time", limits

	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		return "[]byte", limits

	case "numeric", "decimal", "dec", "fixed":
		return "decimal.Decimal", limits

	case "json":
		return "types.JSON[json.RawMessage]", limits

	default:
		return "string", limits
	}
}
