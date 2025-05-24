package parser

// TranslateColumnType converts sqlite database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
// https://sqlite.org/datatype3.html
func TranslateColumnType(dbType string, driver string) string {
	switch dbType {
	case "TINYINT", "INT8":
		return "int8"
	case "SMALLINT", "INT2":
		return "int16"
	case "MEDIUMINT":
		return "int32"
	case "INT", "INTEGER":
		return "int32"
	case "BIGINT":
		return "int64"
	case "UNSIGNED BIG INT":
		return "uint64"
	case "CHARACTER", "VARCHAR", "VARYING CHARACTER", "NCHAR",
		"NATIVE CHARACTER", "NVARCHAR", "TEXT", "CLOB":
		return "string"
	case "BLOB":
		return "[]byte"
	case "FLOAT", "REAL":
		return "float32"
	case "DOUBLE", "DOUBLE PRECISION":
		return "float64"
	case "NUMERIC", "DECIMAL":
		return "decimal.Decimal"
	case "BOOLEAN":
		return "bool"
	case "DATE", "DATETIME", "TIMESTAMP":
		if driver == "libsql" {
			return "types.Time"
		}
		return "time.Time"
	case "JSON":
		return "types.JSON[json.RawMessage]"

	default:
		return "string"
	}
}
