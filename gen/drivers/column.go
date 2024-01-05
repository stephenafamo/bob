package drivers

import (
	"strings"

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
	Generated bool   `json:"generated" yaml:"generated" toml:"generated"`
	AutoIncr  bool   `json:"autoincr" yaml:"autoincr" toml:"autoincr"`

	// DomainName is the domain type name associated to the column. See here:
	// https://www.postgresql.org/docs/16/extend-type-system.html
	DomainName string `json:"domain_name" yaml:"domain_name" toml:"domain_name"`

	Type string `json:"type" yaml:"type" toml:"type"`
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
