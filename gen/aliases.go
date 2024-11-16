package gen

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
)

// initAliases takes the table information from the driver
// and fills in aliases where the user has provided none.
//
// This leaves us with a complete list of Go names for all tables,
// columns, and relationships.
func initAliases[C, I any](a drivers.Aliases, tables drivers.Tables[C, I], relMap Relationships) {
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
