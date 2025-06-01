package drivers

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
)

// Table metadata from the database schema.
type Table[ConstraintExtra, IndexExtra any] struct {
	Key string `yaml:"key" json:"key"`
	// For dbs with real schemas, like Postgres.
	// Example value: "schema_name"."table_name"
	Schema      string                       `yaml:"schema" json:"schema"`
	Name        string                       `yaml:"name" json:"name"`
	Columns     []Column                     `yaml:"columns" json:"columns"`
	Indexes     []Index[IndexExtra]          `yaml:"indexes" json:"indexes"`
	Constraints Constraints[ConstraintExtra] `yaml:"constraints" json:"constraints"`
	Comment     string                       `json:"comment" yaml:"comment"`
}

func (t Table[C, I]) DBTag(c Column) string {
	tag := c.Name
	if t.Constraints.Primary != nil {
		for _, pkc := range t.Constraints.Primary.Columns {
			if pkc == c.Name {
				tag += ",pk"
			}
		}
	}
	if c.Generated {
		tag += ",generated"
	}
	if c.AutoIncr {
		tag += ",autoincr"
	}
	return tag
}

func (t Table[C, I]) NonGeneratedColumns() []Column {
	cols := make([]Column, 0, len(t.Columns))
	for _, c := range t.Columns {
		if c.Generated {
			continue
		}
		cols = append(cols, c)
	}

	return cols
}

func (t Table[C, I]) CanSoftDelete(deleteColumn string) bool {
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

// GetColumn by name. Panics if not found (for use in templates mostly).
func (t Table[C, I]) GetColumn(name string) Column {
	for _, c := range t.Columns {
		if c.Name == name {
			return c
		}
	}

	panic(fmt.Sprintf("could not find column name: %q.%q in %#v", t.Key, name, t.Columns))
}

// Returns true if the table has a unique constraint on exactly these columns
func (t Table[C, I]) HasExactUnique(cols ...string) bool {
	if len(cols) == 0 {
		return false
	}

	// Primary keys are unique
	if t.Constraints.Primary != nil && internal.SliceMatch(t.Constraints.Primary.Columns, cols) {
		return true
	}

	// Check other unique constrints
	for _, u := range t.Constraints.Uniques {
		if internal.SliceMatch(u.Columns, cols) {
			return true
		}
	}

	return false
}

func (t Table[C, I]) RelIsRequired(rel orm.Relationship) bool {
	// The relationship is not required, if its not using foreign keys
	if rel.NeverRequired {
		return false
	}

	firstSide := rel.Sides[0]
	if firstSide.Modify == "to" {
		return false
	}

	for _, colName := range firstSide.FromColumns {
		if !t.GetColumn(colName).Nullable {
			return true
		}
	}

	return false
}

// Used in templates to know if the given table is a join table for this relationship
func (t Table[C, I]) IsJoinTable() bool {
	// Must have exactly 2 foreign keys
	if len(t.Constraints.Foreign) != 2 {
		return false
	}

	// Extract the columns names
	colNames := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		colNames[i] = c.Name
	}

	// All columns must be contained in the foreign keys
	if !internal.AllColsInList(colNames, t.Constraints.Foreign[0].Columns, t.Constraints.Foreign[1].Columns) {
		return false
	}

	// Must have a unique constraint on all columns
	return t.HasExactUnique(colNames...)
}

// Used in templates to know if the given table is a join table for this relationship
func (t Table[C, I]) IsJoinTableForRel(r orm.Relationship, position int) bool {
	if position == 0 || len(r.Sides) < 2 {
		return false
	}

	if position == len(r.Sides) {
		return false
	}

	if t.Key != r.Sides[position-1].To {
		panic(fmt.Sprintf(
			"table name does not match relationship position, expected %s got %s",
			t.Key, r.Sides[position-1].To,
		))
	}

	relevantSides := r.Sides[position-1 : position+1]

	// If the external mappings are not unique, it is not a join table
	if !relevantSides[0].FromUnique || !relevantSides[1].ToUnique {
		return false
	}

	// Extract the columns names
	colNames := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		colNames[i] = c.Name
	}

	if !internal.AllColsInList(
		colNames,
		relevantSides[0].IgnoredColumns[1], relevantSides[0].ToColumns,
		relevantSides[1].IgnoredColumns[0], relevantSides[1].FromColumns,
	) {
		return false
	}

	// These are the columns actually used in the relationship
	// i.e. not ignored
	relevantColumns := append(
		relevantSides[0].ToColumns,
		relevantSides[1].FromColumns...,
	)

	// Must have a unique constraint on all columns
	return t.HasExactUnique(internal.RemoveDuplicates(relevantColumns)...)
}

func (t Table[C, I]) UniqueColPairs() string {
	ret := make([]string, 0, len(t.Constraints.Uniques)+1)

	ret = append(ret, fmt.Sprintf("%#v", t.Constraints.Primary.Columns))
	for _, unique := range t.Constraints.Uniques {
		ret = append(ret, fmt.Sprintf("%#v", unique.Columns))
	}

	return strings.Join(ret, ", ")
}
