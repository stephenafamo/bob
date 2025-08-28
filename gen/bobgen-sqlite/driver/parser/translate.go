package parser

import "strings"

// TranslateColumnType converts sqlite database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
// https://sqlite.org/datatype3.html
func TranslateColumnType(dbType, driver string) string {
	// Some common types
	switch dbType {
	case "NUMERIC", "DECIMAL":
		return "decimal.Decimal"
	case "BOOLEAN":
		return "bool"
	case "DATE", "DATETIME", "TIMESTAMP":
		if driver == "libsql" {
			return "types.Time"
		}
		return "time.Time"
	case "JSON", "JSONB":
		return "types.JSON[json.RawMessage]"
	}

	switch {
	case strings.Contains(dbType, "INT"):
		// Any type with "INT" in it is INTEGER affinity
		// and integers are ALWAYS int64 in SQLite
		return "int64"

	case strings.Contains(dbType, "CHAR"),
		strings.Contains(dbType, "CLOB"),
		strings.Contains(dbType, "TEXT"):
		return "string"

	case strings.Contains(dbType, "BLOB"):
		return "[]byte"

	case strings.Contains(dbType, "REAL"),
		strings.Contains(dbType, "FLOA"),
		strings.Contains(dbType, "DOUB"): //nolint:misspell
		// All floats are float64 in SQLite
		return "float64"

	default:
		// Even if the default affinity is NUMERIC, we map it to "string"
		// because in SQLite NUMERIC can use all storage classes
		return "string"

	}
}
