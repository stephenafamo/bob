package drivers

import (
	"fmt"
)

// Table metadata from the database schema.
type Table struct {
	Key string `yaml:"key" json:"key"`
	// For dbs with real schemas, like Postgres.
	// Example value: "schema_name"."table_name"
	Schema  string   `yaml:"schema" json:"schema"`
	Name    string   `yaml:"name" json:"name"`
	Columns []Column `yaml:"columns" json:"columns"`

	Constraints Constraints `yaml:"constraints" json:"constraints"`
}

type Constraints struct {
	Primary *PrimaryKey  `yaml:"primary" json:"primary"`
	Foreign []ForeignKey `yaml:"foreign" json:"foreign"`
	Uniques []Constraint `yaml:"uniques" json:"uniques"`
}

// GetTable by name. Panics if not found (for use in templates mostly).
func GetTable(tables []Table, name string) Table {
	for _, t := range tables {
		if t.Key == name {
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

	panic(fmt.Sprintf("could not find column name: %q.%q in %#v", t.Key, name, t.Columns))
}

func (t Table) NonGeneratedColumns() []Column {
	cols := make([]Column, 0, len(t.Columns))
	for _, c := range t.Columns {
		if c.Generated {
			continue
		}
		cols = append(cols, c)
	}

	return cols
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

type Filter struct {
	Only   []string
	Except []string
}

type ColumnFilter map[string]Filter

func ParseTableFilter(only, except map[string][]string) Filter {
	var filter Filter
	for name := range only {
		filter.Only = append(filter.Only, name)
	}

	for name, cols := range except {
		// If they only want to exclude some columns, then we don't want to exclude the whole table
		if len(cols) == 0 {
			filter.Except = append(filter.Except, name)
		}
	}

	return filter
}

func ParseColumnFilter(tables []string, only, except map[string][]string) ColumnFilter {
	global := Filter{
		Only:   only["*"],
		Except: except["*"],
	}

	colFilter := make(ColumnFilter, len(tables))
	for _, t := range tables {
		colFilter[t] = Filter{
			Only:   append(global.Only, only[t]...),
			Except: append(global.Except, except[t]...),
		}
	}
	return colFilter
}
