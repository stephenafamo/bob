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
type Aliases struct {
	Tables map[string]TableAlias `yaml:"tables,omitempty" toml:"tables,omitempty" json:"tables,omitempty"`
}

// TableAlias defines the spellings for a table name in Go
type TableAlias struct {
	UpPlural     string `yaml:"up_plural,omitempty" toml:"up_plural,omitempty" json:"up_plural,omitempty"`
	UpSingular   string `yaml:"up_singular,omitempty" toml:"up_singular,omitempty" json:"up_singular,omitempty"`
	DownPlural   string `yaml:"down_plural,omitempty" toml:"down_plural,omitempty" json:"down_plural,omitempty"`
	DownSingular string `yaml:"down_singular,omitempty" toml:"down_singular,omitempty" json:"down_singular,omitempty"`

	Columns       map[string]string `yaml:"columns,omitempty" toml:"columns,omitempty" json:"columns,omitempty"`
	Relationships map[string]string `yaml:"relationships,omitempty" toml:"relationships,omitempty" json:"relationships,omitempty"`
}

// FillAliases takes the table information from the driver
// and fills in aliases where the user has provided none.
//
// This leaves us with a complete list of Go names for all tables,
// columns, and relationships.
func FillAliases(a *Aliases, tables []drivers.Table) {
	if a.Tables == nil {
		a.Tables = make(map[string]TableAlias)
	}

	for _, t := range tables {
		table := a.Tables[t.Key]
		cleanKey := strings.ReplaceAll(t.Key, ".", "_")

		if len(table.UpPlural) == 0 {
			table.UpPlural = strmangle.TitleCase(strmangle.Plural(cleanKey))
		}
		if len(table.UpSingular) == 0 {
			table.UpSingular = strmangle.TitleCase(strmangle.Singular(cleanKey))
		}
		if len(table.DownPlural) == 0 {
			table.DownPlural = strmangle.CamelCase(strmangle.Plural(cleanKey))
		}
		if len(table.DownSingular) == 0 {
			table.DownSingular = strmangle.CamelCase(strmangle.Singular(cleanKey))
		}

		if table.Columns == nil {
			table.Columns = make(map[string]string)
		}
		if table.Relationships == nil {
			table.Relationships = make(map[string]string)
		}

		for _, c := range t.Columns {
			if _, ok := table.Columns[c.Name]; !ok {
				table.Columns[c.Name] = strmangle.TitleCase(c.Name)
			}

			r, _ := utf8.DecodeRuneInString(table.Columns[c.Name])
			if unicode.IsNumber(r) {
				table.Columns[c.Name] = "C" + table.Columns[c.Name]
			}
		}

		computed := relAlias(t)
		for _, rel := range t.Relationships {
			if _, ok := table.Relationships[rel.Name]; !ok {
				table.Relationships[rel.Name] = computed[rel.Name]
			}
		}

		a.Tables[t.Key] = table
	}
}

// Table gets a table alias, panics if not found.
func (a Aliases) Table(table string) TableAlias {
	t, ok := a.Tables[table]
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
