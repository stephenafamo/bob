package gen

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
)

// Aliases defines aliases for the generation run
type Aliases struct {
	Tables map[string]TableAlias `toml:"tables,omitempty" json:"tables,omitempty"`
}

// TableAlias defines the spellings for a table name in Go
type TableAlias struct {
	UpPlural     string `toml:"up_plural,omitempty" json:"up_plural,omitempty"`
	UpSingular   string `toml:"up_singular,omitempty" json:"up_singular,omitempty"`
	DownPlural   string `toml:"down_plural,omitempty" json:"down_plural,omitempty"`
	DownSingular string `toml:"down_singular,omitempty" json:"down_singular,omitempty"`

	Columns       map[string]string `toml:"columns,omitempty" json:"columns,omitempty"`
	Relationships map[string]string `toml:"relationships,omitempty" json:"relationships,omitempty"`
}

// RelationshipAlias defines the naming for both sides of
// a foreign key.
type RelationshipAlias struct {
	Local   string `toml:"local,omitempty" json:"local,omitempty"`
	Foreign string `toml:"foreign,omitempty" json:"foreign,omitempty"`
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

		if len(table.UpPlural) == 0 {
			table.UpPlural = strmangle.TitleCase(strmangle.Plural(t.Name))
		}
		if len(table.UpSingular) == 0 {
			table.UpSingular = strmangle.TitleCase(strmangle.Singular(t.Name))
		}
		if len(table.DownPlural) == 0 {
			table.DownPlural = strmangle.CamelCase(strmangle.Plural(t.Name))
		}
		if len(table.DownSingular) == 0 {
			table.DownSingular = strmangle.CamelCase(strmangle.Singular(t.Name))
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
