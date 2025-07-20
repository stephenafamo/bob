package gen

import (
	"fmt"
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
	// Track used table-level aliases
	upAliases := make(map[string]string)
	downAliases := make(map[string]string)

	// Track used column and relationship aliases per table
	columnAliases := make(map[string]map[string]string)
	relationshipAliases := make(map[string]map[string]string)

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

		// Check aliases clash, panic if catch
		checkClashAliases := func(kind string, value string, used map[string]string) {
			if other, ok := used[value]; ok {
				panic(fmt.Sprintf(
					"Alias clash: %s '%s' used by both '%s' and '%s'.\nSuggestion: Override in schema.yml:\n  - name: %s\n    alias:\n      %s: <custom_alias>\n",
					kind, value, other, t.Key, t.Key, strings.ToLower(kind),
				))
			}

			used[value] = t.Key
		}

		checkClashAliases("UpSingular", tableAlias.UpSingular, upAliases)
		checkClashAliases("UpPlural", tableAlias.UpPlural, upAliases)
		checkClashAliases("DownSingular", tableAlias.DownSingular, downAliases)
		checkClashAliases("DownPlural", tableAlias.DownPlural, downAliases)

		if tableAlias.Columns == nil {
			tableAlias.Columns = make(map[string]string)
		}

		if columnAliases[t.Key] == nil {
			columnAliases[t.Key] = make(map[string]string)
		}

		if tableAlias.Relationships == nil {
			tableAlias.Relationships = make(map[string]string)
		}

		if relationshipAliases[t.Key] == nil {
			relationshipAliases[t.Key] = make(map[string]string)
		}

		for _, c := range t.Columns {
			if _, ok := tableAlias.Columns[c.Name]; !ok {
				tableAlias.Columns[c.Name] = strmangle.TitleCase(c.Name)
			}

			colAlias := tableAlias.Columns[c.Name]

			r, _ := utf8.DecodeRuneInString(colAlias)
			if unicode.IsNumber(r) {
				colAlias = "C" + colAlias
				tableAlias.Columns[c.Name] = colAlias
			}

			// Check column existed
			if _, existed := columnAliases[t.Key][colAlias]; existed {
				panic(fmt.Sprintf("Column alias clash: '%s' used more than once in table '%s'", colAlias, t.Key))
			}

			columnAliases[t.Key][colAlias] = c.Name
		}

		tableRels := relMap[t.Key]
		computed := relAlias(tableRels)
		for _, rel := range tableRels {
			if _, ok := tableAlias.Relationships[rel.Name]; !ok {
				tableAlias.Relationships[rel.Name] = computed[rel.Name]
			}

			relAlias := tableAlias.Relationships[rel.Name]

			// Check relationship alias existed
			if _, existed := relationshipAliases[t.Key][relAlias]; existed {
				panic(fmt.Sprintf("Relationship alias clash: '%s' used more than once in table '%s'", relAlias, t.Key))
			}
			relationshipAliases[t.Key][relAlias] = rel.Name
		}

		a[t.Key] = tableAlias
	}
}
