package drivers

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/orm"
)

// Table metadata from the database schema.
type Table struct {
	Key string `json:"key"`
	// For dbs with real schemas, like Postgres.
	// Example value: "schema_name"."table_name"
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
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
		if t.Key == r.Foreign() {
			fTable = t
			break
		}
	}

	// No foreign table matched
	if fTable.Key == "" {
		return orm.Relationship{}
	}

	toMatch := r.Name
	if r.Local() == r.Foreign() {
		hadSuffix := false
		toMatch, hadSuffix = strings.CutSuffix(r.Name, SelfJoinSuffix)
		if !hadSuffix {
			toMatch += SelfJoinSuffix
		}
	}

	for _, r2 := range fTable.Relationships {
		if toMatch == r2.Name {
			return r2
		}
	}

	return orm.Relationship{}
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

	for name := range except {
		filter.Except = append(filter.Except, name)
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
