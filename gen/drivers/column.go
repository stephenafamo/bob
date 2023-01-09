package drivers

import (
	"strings"

	"github.com/stephenafamo/bob/gen/importers"
	"github.com/volatiletech/strmangle"
)

// Column holds information about a database column.
// Types are Go types, converted by TranslateColumnType.
type Column struct {
	Name      string `json:"name" yaml:"name" toml:"name"`
	DBType    string `json:"db_type" yaml:"db_type" toml:"db_type"`
	Default   string `json:"default" yaml:"default" toml:"default"`
	Comment   string `json:"comment" yaml:"comment" toml:"comment"`
	Nullable  bool   `json:"nullable" yaml:"nullable" toml:"nullable"`
	Unique    bool   `json:"unique" yaml:"unique" toml:"unique"`
	Generated bool   `json:"generated" yaml:"generated" toml:"generated"`

	Type    string         `json:"type" yaml:"type" toml:"type"`
	Imports importers.List `json:"imports" yaml:"imports" toml:"imports"`

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

// ColumnNames of the columns.
func ColumnNames(cols []Column) []string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}

	return names
}

// ColumnDBTypes of the columns.
func ColumnDBTypes(cols []Column) map[string]string {
	types := map[string]string{}

	for _, c := range cols {
		types[strmangle.TitleCase(c.Name)] = c.DBType
	}

	return types
}

// ColumnsFromList takes a whitelist or blacklist and returns
// the columns for a given table.
func ColumnsFromList(list []string, tablename string) []string {
	if len(list) == 0 {
		return nil
	}

	var columns []string
	for _, i := range list {
		splits := strings.Split(i, ".")

		if len(splits) != 2 {
			continue
		}

		if splits[0] == tablename || splits[0] == "*" {
			columns = append(columns, splits[1])
		}
	}

	return columns
}
