package drivers

import (
	"fmt"
)

// Aliases defines aliases for the generation run
type Aliases map[string]TableAlias

// TableAlias defines the spellings for a table name in Go
type TableAlias struct {
	UpPlural     string `yaml:"up_plural,omitempty" json:"up_plural,omitempty"`
	UpSingular   string `yaml:"up_singular,omitempty" json:"up_singular,omitempty"`
	DownPlural   string `yaml:"down_plural,omitempty" json:"down_plural,omitempty"`
	DownSingular string `yaml:"down_singular,omitempty" json:"down_singular,omitempty"`

	Columns       map[string]string `yaml:"columns,omitempty" json:"columns,omitempty"`
	Relationships map[string]string `yaml:"relationships,omitempty" json:"relationships,omitempty"`
}

// Table gets a table alias, panics if not found.
func (a Aliases) Table(table string) TableAlias {
	t, ok := a[table]
	if !ok {
		panic("could not find table aliases for: " + table)
	}

	return t
}

// Column get's a column's aliased name, panics if not found.
func (t TableAlias) Column(column string) string {
	c, ok := t.Columns[column]
	if !ok {
		fmt.Printf("TableAlias: %#v\n", t)
		panic(fmt.Sprintf("could not find column alias for: %s.%s", t.UpSingular, column))
	}

	return c
}

// Relationship looks up a relationship, panics if not found.
func (t TableAlias) Relationship(fkey string) string {
	r, ok := t.Relationships[fkey]
	if !ok {
		panic(fmt.Sprintf("could not find relationship alias for: %s.%s", t.UpSingular, fkey))
	}

	return r
}
