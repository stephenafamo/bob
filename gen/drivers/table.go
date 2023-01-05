package drivers

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/orm"
)

// Table metadata from the database schema.
type Table struct {
	Name string `json:"name"`
	// For dbs with real schemas, like Postgres.
	// Example value: "schema_name"."table_name"
	Schema  string   `json:"schema"`
	Columns []Column `json:"columns"`

	PKey    *PrimaryKey  `json:"p_key"`
	FKeys   []ForeignKey `json:"foreign_keys"`
	Uniques []Constraint `json:"unique"`

	IsJoinTable bool `json:"is_join_table"`

	Relationships []orm.Relationship `json:"relationship"`
}

// GetTable by name. Panics if not found (for use in templates mostly).
func GetTable(tables []Table, name string) Table {
	for _, t := range tables {
		if t.Name == name {
			return t
		}
	}

	panic(fmt.Sprintf("could not find table name: %s", name))
}

// GetColumn by name. Panics if not found (for use in templates mostly).
func (t Table) GetColumn(name string) Column {
	for _, c := range t.Columns {
		if c.Name == name {
			return c
		}
	}

	panic(fmt.Sprintf("could not find column name: %s", name))
}

func (t Table) CanSoftDelete(deleteColumn string) bool {
	if deleteColumn == "" {
		deleteColumn = "deleted_at"
	}

	for _, column := range t.Columns {
		if column.Name == deleteColumn && column.Type == "null.Time" {
			return true
		}
	}
	return false
}

// GetRelationshipInverse returns the Relationship of the other side
func (t Table) GetRelationshipInverse(tables []Table, r orm.Relationship) orm.Relationship {
	var fTable Table
	for _, t := range tables {
		if t.Name == r.Foreign() {
			fTable = t
			break
		}
	}

	// No foreign table matched
	if fTable.Name == "" {
		return orm.Relationship{}
	}

	for _, r2 := range fTable.Relationships {
		if r.Name == r2.Name {
			return r2
		}
	}

	return orm.Relationship{}
}

// TablesFromList takes a whitelist or blacklist and returns
// the table names.
func TablesFromList(list []string) []string {
	if len(list) == 0 {
		return nil
	}

	var tables []string
	for _, i := range list {
		splits := strings.Split(i, ".")

		if len(splits) == 1 {
			tables = append(tables, splits[0])
		}
	}

	return tables
}

type Filter struct {
	Include []string
	Exclude []string
}

type ColumnFilter map[string]Filter

// This takes a list of table names with the includes and excludes
func ParseColumnFilter(tables, includes, excludes []string) ColumnFilter {
	colFilter := make(ColumnFilter, len(tables)+1)

	if len(tables) == 0 {
		return colFilter
	}
	colFilter["*"] = Filter{
		Include: columnsFromList(includes, "*"),
		Exclude: columnsFromList(excludes, "*"),
	}

	for _, t := range tables {
		colFilter[t] = Filter{
			Include: columnsFromList(includes, t),
			Exclude: columnsFromList(excludes, t),
		}
	}

	return colFilter
}

// like ColumnsFromList, but does not include wildcard columns
func columnsFromList(list []string, tablename string) []string {
	if len(list) == 0 {
		return nil
	}

	var columns []string
	for _, i := range list {
		splits := strings.Split(i, ".")

		if len(splits) != 2 {
			continue
		}

		if splits[0] == tablename {
			columns = append(columns, splits[1])
		}
	}

	return columns
}
