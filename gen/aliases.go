package gen

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
)

// Aliases defines aliases for the generation run
type Aliases map[string]TableAlias

// TableAlias defines the spellings for a table name in Go
type TableAlias struct {
	UpPlural     string `yaml:"up_plural,omitempty" toml:"up_plural,omitempty" json:"up_plural,omitempty"`
	UpSingular   string `yaml:"up_singular,omitempty" toml:"up_singular,omitempty" json:"up_singular,omitempty"`
	DownPlural   string `yaml:"down_plural,omitempty" toml:"down_plural,omitempty" json:"down_plural,omitempty"`
	DownSingular string `yaml:"down_singular,omitempty" toml:"down_singular,omitempty" json:"down_singular,omitempty"`

	Columns       map[string]string `yaml:"columns,omitempty" toml:"columns,omitempty" json:"columns,omitempty"`
	Relationships map[string]string `yaml:"relationships,omitempty" toml:"relationships,omitempty" json:"relationships,omitempty"`
}

// initAliases takes the table information from the driver
// and fills in aliases where the user has provided none.
//
// This leaves us with a complete list of Go names for all tables,
// columns, and relationships.
func initAliases(a Aliases, tables []drivers.Table, relMap Relationships) {
	for _, t := range tables {
		tableAlias := a[t.Key]
		cleanKey := strings.ReplaceAll(t.Key, ".", "_")

		if len(tableAlias.UpPlural) == 0 {
			tableAlias.UpPlural = strmangle.TitleCase(strmangle.Plural(cleanKey))
		}
		if len(tableAlias.UpSingular) == 0 {
			tableAlias.UpSingular = strmangle.TitleCase(strmangle.Singular(cleanKey))
		}
		if len(tableAlias.DownPlural) == 0 {
			tableAlias.DownPlural = strmangle.CamelCase(strmangle.Plural(cleanKey))
		}
		if len(tableAlias.DownSingular) == 0 {
			tableAlias.DownSingular = strmangle.CamelCase(strmangle.Singular(cleanKey))
		}

		if tableAlias.Columns == nil {
			tableAlias.Columns = make(map[string]string)
		}
		if tableAlias.Relationships == nil {
			tableAlias.Relationships = make(map[string]string)
		}

		for _, c := range t.Columns {
			if _, ok := tableAlias.Columns[c.Name]; !ok {
				tableAlias.Columns[c.Name] = strmangle.TitleCase(c.Name)
			}

			r, _ := utf8.DecodeRuneInString(tableAlias.Columns[c.Name])
			if unicode.IsNumber(r) {
				tableAlias.Columns[c.Name] = "C" + tableAlias.Columns[c.Name]
			}
		}

		tableRels := relMap[t.Key]
		computed := relAlias(tableRels)
		for _, rel := range tableRels {
			if _, ok := tableAlias.Relationships[rel.Name]; !ok {
				tableAlias.Relationships[rel.Name] = computed[rel.Name]
			}
		}

		a[t.Key] = tableAlias
	}
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
